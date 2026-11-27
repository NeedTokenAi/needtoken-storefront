package application

import (
	"testing"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/modules/captcha/contract"
	settingssecurity "github.com/dujiao-next/internal/modules/settings/schema/security"
)

type settingReaderStub struct {
	setting settingssecurity.CaptchaSetting
}

func (s settingReaderStub) GetCaptchaSetting(config.CaptchaConfig) (settingssecurity.CaptchaSetting, error) {
	return s.setting, nil
}

type turnstileStub struct {
	cfg      settingssecurity.CaptchaTurnstileSetting
	token    string
	clientIP string
}

func (s *turnstileStub) Verify(cfg settingssecurity.CaptchaTurnstileSetting, token, clientIP string) error {
	s.cfg = cfg
	s.token = token
	s.clientIP = clientIP
	return nil
}

func TestVerifyTurnstileDelegatesToVerifier(t *testing.T) {
	setting := settingssecurity.CaptchaSetting{
		Provider: constants.CaptchaProviderTurnstile,
		Scenes: settingssecurity.CaptchaSceneSetting{
			Login: true,
		},
		Turnstile: settingssecurity.CaptchaTurnstileSetting{
			SecretKey: "secret",
			VerifyURL: "https://captcha.example/verify",
			TimeoutMS: 2000,
		},
	}
	verifier := &turnstileStub{}
	service := NewService(settingReaderStub{setting: setting}, config.CaptchaConfig{}, verifier)

	err := service.Verify(constants.CaptchaSceneLogin, contract.VerifyPayload{TurnstileToken: " token-1 "}, " 127.0.0.1 ")
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if verifier.token != "token-1" || verifier.clientIP != "127.0.0.1" || verifier.cfg.SecretKey != "secret" {
		t.Fatalf("turnstile input mismatch: token=%q ip=%q cfg=%#v", verifier.token, verifier.clientIP, verifier.cfg)
	}
}

func TestVerifyImageRequiresChallengePayload(t *testing.T) {
	setting := settingssecurity.CaptchaSetting{
		Provider: constants.CaptchaProviderImage,
		Scenes: settingssecurity.CaptchaSceneSetting{
			Login: true,
		},
	}
	service := NewService(settingReaderStub{setting: setting}, config.CaptchaConfig{}, &turnstileStub{})
	if err := service.Verify(constants.CaptchaSceneLogin, contract.VerifyPayload{}, ""); err != contract.ErrRequired {
		t.Fatalf("empty image captcha error got %v want ErrRequired", err)
	}
}

func TestVerifySkipsDisabledScene(t *testing.T) {
	setting := settingssecurity.CaptchaSetting{Provider: constants.CaptchaProviderTurnstile}
	service := NewService(settingReaderStub{setting: setting}, config.CaptchaConfig{}, nil)
	if err := service.Verify(constants.CaptchaSceneLogin, contract.VerifyPayload{}, ""); err != nil {
		t.Fatalf("disabled scene must skip captcha, got %v", err)
	}
}

func TestTurnstileRejectsImageCaptchaFallback(t *testing.T) {
	setting := settingssecurity.CaptchaSetting{
		Provider: constants.CaptchaProviderTurnstile,
		Scenes: settingssecurity.CaptchaSceneSetting{
			Login: true,
		},
		Image: settingssecurity.CaptchaImageSetting{
			Length:        5,
			Width:         240,
			Height:        80,
			NoiseCount:    2,
			ShowLine:      2,
			ExpireSeconds: 300,
			MaxStore:      1024,
		},
		Turnstile: settingssecurity.CaptchaTurnstileSetting{
			SiteKey:   "site-key",
			SecretKey: "secret",
			VerifyURL: "https://captcha.example/verify",
			TimeoutMS: 2000,
		},
	}
	verifier := &turnstileStub{}
	service := NewService(settingReaderStub{setting: setting}, config.CaptchaConfig{}, verifier)

	// Turnstile 模式下禁止签发图片验证码，防止绕过 Cloudflare 验证
	if challenge, err := service.GenerateImageChallenge(); err != contract.ErrConfigInvalid {
		t.Fatalf("GenerateImageChallenge in turnstile mode got (%#v, %v), want ErrConfigInvalid", challenge, err)
	}

	// 空载荷必须拒绝
	if err := service.Verify(constants.CaptchaSceneLogin, contract.VerifyPayload{}, "127.0.0.1"); err != contract.ErrRequired {
		t.Fatalf("empty payload error got %v want ErrRequired", err)
	}

	// 即使图片验证码存储中存在有效答案，Turnstile 模式下提交图片验证码载荷也必须被拒绝
	store := service.ensureImageStore(setting)
	if err := store.Set("fallback-id", "Ab12C"); err != nil {
		t.Fatalf("seed image store failed: %v", err)
	}
	if err := service.Verify(constants.CaptchaSceneLogin, contract.VerifyPayload{
		CaptchaID:   "fallback-id",
		CaptchaCode: "Ab12C",
	}, "127.0.0.1"); err != contract.ErrRequired {
		t.Fatalf("image captcha fallback in turnstile mode got %v want ErrRequired", err)
	}

	// 提交 Turnstile token 正常放行给 Turnstile verifier
	if err := service.Verify(constants.CaptchaSceneLogin, contract.VerifyPayload{
		TurnstileToken: "valid-turnstile-token",
	}, "127.0.0.1"); err != nil {
		t.Fatalf("valid turnstile token failed: %v", err)
	}
	if verifier.token != "valid-turnstile-token" {
		t.Fatalf("verifier token got %q want %q", verifier.token, "valid-turnstile-token")
	}
}
