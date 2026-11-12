package queue

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type commandLog struct {
	mu   sync.Mutex
	args [][]string
}

func (l *commandLog) Commands() [][]string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([][]string, len(l.args))
	copy(out, l.args)
	return out
}

func (l *commandLog) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return nil, errors.New("staging test does not open redis")
	}
}

func (l *commandLog) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		parts := make([]string, 0, len(cmd.Args()))
		for _, arg := range cmd.Args() {
			parts = append(parts, fmt.Sprint(arg))
		}
		l.mu.Lock()
		l.args = append(l.args, parts)
		l.mu.Unlock()
		return errors.New("staging test refused the redis write")
	}
}

func (l *commandLog) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		for _, cmd := range cmds {
			if err := l.ProcessHook(nil)(ctx, cmd); err != nil {
				return err
			}
		}
		return errors.New("staging test refused the redis write")
	}
}

type recordingOpt struct {
	log *commandLog
}

func (o recordingOpt) MakeRedisClient() any {
	client := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: time.Millisecond,
		MaxRetries:  -1,
	})
	client.AddHook(o.log)
	return client
}

func TestStagingEnqueueWritesNoUnprefixedQueueKey(t *testing.T) {
	log := &commandLog{}
	client, err := NewClient(&config.QueueConfig{
		Enabled: true,
		Host:    "production.upstash.example",
		Port:    6379,
	}, WithKeyPrefix("staging:"), WithEnqueueAllowed(false), withRedisConn(recordingOpt{log: log}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	require.Equal(t, "staging:default", client.defaultQueue)
	require.Equal(t, "asynq:{staging:default}:", asynqQueueKeyPrefix(client.defaultQueue))
	require.NotEqual(t, "asynq:{default}:", asynqQueueKeyPrefix(client.defaultQueue))
	require.Equal(t, "default", QueueName("dj", DefaultQueue))

	attempts := []error{
		client.EnqueueOrderStatusEmail(OrderStatusEmailPayload{OrderID: 9, Status: "paid"}),
		client.EnqueueOrderAutoFulfill(OrderAutoFulfillPayload{OrderID: 9}),
		client.EnqueueAffiliateOrderPaid(AffiliateOrderPaidPayload{OrderID: 9}),
		client.EnqueueOrderTimeoutCancel(OrderTimeoutCancelPayload{OrderID: 9}, time.Minute),
		client.EnqueueWalletRechargeExpire(WalletRechargeExpirePayload{PaymentID: 9}, time.Minute),
		client.EnqueueNotificationDispatch(NotificationDispatchPayload{EventType: "order.paid", BizType: "order", BizID: 9}),
		client.EnqueueProcurementSubmit(ProcurementSubmitPayload{}),
		client.EnqueueProcurementPollStatus(ProcurementPollStatusPayload{}, time.Minute),
		client.EnqueueDownstreamCallback(DownstreamCallbackPayload{}),
		client.EnqueueReconciliationRun(ReconciliationRunPayload{}),
		client.EnqueueBotNotify(BotNotifyPayload{}),
		client.EnqueueTelegramBroadcast(TelegramBroadcastPayload{}),
	}
	for _, err := range attempts {
		require.ErrorIs(t, err, ErrEnqueueRefused)
	}

	for _, command := range log.Commands() {
		for _, arg := range command {
			require.NotContains(t, arg, "asynq:{default}")
			require.NotEqual(t, "asynq:queues", arg)
			if strings.Contains(arg, "asynq:") {
				require.Contains(t, arg, "staging")
			}
		}
	}
	require.Empty(t, log.Commands())
}
