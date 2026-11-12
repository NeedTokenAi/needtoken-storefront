package app

import (
	"errors"
	"time"

	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/app/httpserver"
	"github.com/dujiao-next/internal/app/jobs"
	jobconsumer "github.com/dujiao-next/internal/app/jobs/consumer"
	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/platform/cloudflare"
	"github.com/dujiao-next/internal/workflows/paidfulfillment"
)

// BuildRunner 构建服务运行器
func BuildRunner(cfg *config.Config, mode string) (*Runner, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}
	resolved, err := cloudflare.ResolveMode(cfg.Cloudflare.Enabled, mode)
	if err != nil {
		return nil, err
	}
	if err := cloudflare.Validate(cfg); err != nil {
		return nil, err
	}
	if cfg.Cloudflare.Enabled {
		mode = resolved
	}

	dependencies, err := container.NewContainer(cfg)
	if err != nil {
		return nil, err
	}

	var services []Service
	if cfg.Cloudflare.Enabled {
		dependencies.CloudflareRuntime = cloudflare.NewRuntime(cfg.Cloudflare.WriterFence)
	}

	// 初始化 HTTP 服务
	if mode == ModeAll || mode == ModeAPI {
		engine := httpserver.SetupRouter(cfg, dependencies)
		addr := cfg.Server.Host + ":" + cfg.Server.Port
		httpService := NewHTTPService(addr, engine)
		services = append(services, httpService)
	}

	// 初始化 Worker 服务
	if mode == ModeWorker && !cfg.Queue.Enabled {
		return nil, errors.New("queue disabled")
	}
	if cfg.Queue.Enabled && (mode == ModeAll || mode == ModeWorker) {
		consumer := jobconsumer.New(dependencies)
		workerService, err := jobs.NewService(&cfg.Queue, consumer, !cfg.Cloudflare.Enabled)
		if err != nil {
			return nil, err
		}
		if dependencies.CloudflareRuntime != nil {
			workerService.NotifyRunning(dependencies.CloudflareRuntime.SetDispatcher)
		}
		services = append(services, workerService)
	}

	// 如果没有服务被启动（例如模式错误或配置导致都没起），应该报错或至少打日志
	if len(services) == 0 {
		return nil, errors.New("no services initialized (check mode and config)")
	}

	services = append(services, &paidFulfillmentService{payments: paidfulfillment.New(dependencies.PaymentStore, dependencies.ProcurementOrderService, dependencies.PaymentService)})
	return NewRunner(services...), nil
}

// Run 应用启动入口
func Run(opts Options) error {
	opts = normalizeOptions(opts)
	if opts.Config == nil {
		return errors.New("config is nil")
	}
	if opts.Config.Cloudflare.Enabled && (opts.BaseContext == nil || opts.BaseContext.Err() != nil) {
		return errors.New("cloudflare profile: background start requires a live isolated writer permit; this lease does not stop the existing Cloud Run writer")
	}

	runner, err := BuildRunner(opts.Config, opts.Mode)
	if err != nil {
		return err
	}

	addr := opts.Config.Server.Host + ":" + opts.Config.Server.Port
	opts.Logger.Infow("app_start", "addr", addr, "mode", opts.Mode)
	if opts.Config.Cloudflare.Enabled && opts.ShutdownTimeout == 10*time.Second {
		opts.ShutdownTimeout = cloudflare.DrainTimeout
	}
	return RunWithOptions(runner, opts)
}
