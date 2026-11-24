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

func TestTurnstileAllowsFallbackToImageCaptcha(t *testing.T) {
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
	service := NewService(settingReaderStub{setting: setting}, config.CaptchaConfig{}, &turnstileStub{})

	// Turnstile 模式下允许签发降级图片验证码
	challenge, err := service.GenerateImageChallenge()
	if err != nil {
		t.Fatalf("GenerateImageChallenge in turnstile mode failed: %v", err)
	}
	if challenge == nil || challenge.CaptchaID == "" || challenge.ImageBase64 == "" {
		t.Fatalf("unexpected image challenge: %#v", challenge)
	}

	// 双空必须拒绝
	if err := service.Verify(constants.CaptchaSceneLogin, contract.VerifyPayload{}, "127.0.0.1"); err != contract.ErrRequired {
		t.Fatalf("empty payload error got %v want ErrRequired", err)
	}

	// 写入确定性验证码答案验证降级成功与一次性防重放
	store := service.ensureImageStore(setting)
	if err := store.Set("fallback-id", "Ab12C"); err != nil {
		t.Fatalf("seed image store failed: %v", err)
	}
	if err := service.Verify(constants.CaptchaSceneLogin, contract.VerifyPayload{
		CaptchaID:   "fallback-id",
		CaptchaCode: "wrong",
	}, "127.0.0.1"); err != contract.ErrInvalid {
		t.Fatalf("wrong fallback code got %v want ErrInvalid", err)
	}
	// 上一步 verify(clear=true) 已消耗旧 id，重新写入验证正确码通过且不可重放
	if err := store.Set("fallback-id-2", "Xy98Z"); err != nil {
		t.Fatalf("seed image store 2 failed: %v", err)
	}
	if err := service.Verify(constants.CaptchaSceneLogin, contract.VerifyPayload{
		CaptchaID:   "fallback-id-2",
		CaptchaCode: "Xy98Z",
	}, "127.0.0.1"); err != nil {
		t.Fatalf("valid fallback image captcha failed: %v", err)
	}
	if err := service.Verify(constants.CaptchaSceneLogin, contract.VerifyPayload{
		CaptchaID:   "fallback-id-2",
		CaptchaCode: "Xy98Z",
	}, "127.0.0.1"); err != contract.ErrInvalid {
		t.Fatalf("replayed fallback image captcha got %v want ErrInvalid", err)
	}
}
