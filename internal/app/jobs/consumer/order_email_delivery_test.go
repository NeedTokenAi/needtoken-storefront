package consumer

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"

	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	fulfillmentdomain "github.com/dujiao-next/internal/modules/fulfillment/domain"
	notificationsmtp "github.com/dujiao-next/internal/modules/notification/infrastructure/smtp"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	"github.com/dujiao-next/internal/queue"
)

// captureSMTP accepts one plain SMTP session and returns the DATA it received.
func captureSMTP(t *testing.T) (port int, data <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	out := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		write := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
		write("220 test")
		var body strings.Builder
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			if inData {
				if line == ".\r\n" {
					inData = false
					out <- body.String()
					write("250 ok")
					continue
				}
				body.WriteString(line)
				continue
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				write("250 test")
			case cmd == "DATA":
				inData = true
				write("354 go")
			case cmd == "QUIT":
				write("221 bye")
				return
			default:
				write("250 ok")
			}
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, out
}

func TestOrderDeliveredEmailOmitsDeliveryContent(t *testing.T) {
	const secret = "REDEEM-CODE-DO-NOT-EMAIL"
	for _, payload := range []string{secret, strings.Repeat(secret+"\n", 2000)} {
		port, data := captureSMTP(t)
		cfg := config.EmailConfig{Enabled: true, Host: "127.0.0.1", Port: port, From: "shop@example.com"}
		order := &orderdomain.Order{
			ID: 9, OrderNo: "DJ-ORDER-009", GuestEmail: "buyer@example.com", GuestLocale: "zh-CN",
			Currency: "USD", Status: constants.OrderStatusDelivered,
			Fulfillment: &fulfillmentdomain.Fulfillment{Payload: payload},
			Children: []orderdomain.Order{{OrderNo: "DJ-ORDER-009-1",
				Fulfillment: &fulfillmentdomain.Fulfillment{Payload: secret}}},
		}
		task, err := queue.NewOrderStatusEmailTask(queue.OrderStatusEmailPayload{
			OrderID: order.ID, Status: constants.OrderStatusDelivered,
		})
		if err != nil {
			t.Fatal(err)
		}
		consumer := &Consumer{
			Container:   &container.Container{EmailSender: notificationsmtp.New(&cfg)},
			orderReader: orderStatusEmailWorkerOrderRepoStub{order: order},
		}
		if err := consumer.handleOrderStatusEmail(context.Background(), task); err != nil {
			t.Fatalf("payload len %d: send: %v", len(payload), err)
		}
		msg := <-data
		if !strings.Contains(msg, "DJ-ORDER-009") {
			t.Fatalf("expected the order email to be sent, got %q", msg)
		}
		if strings.Contains(msg, secret) || strings.Contains(strings.ToLower(msg), "attachment") {
			t.Fatalf("payload len %d: email leaks delivery content:\n%s", len(payload), msg)
		}
	}
}
