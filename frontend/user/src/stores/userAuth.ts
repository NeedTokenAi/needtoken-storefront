import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { userAuthAPI } from '../api'
import { resetApiAccountLinkCache, triggerSilentApiAccountLinkEnsure } from '../templates/needtoken/composables/useApiAccountLink'

export const useUserAuthStore = defineStore('user-auth', () => {
    const router = useRouter()

    const token = ref<string>(localStorage.getItem('user_token') || '')
    const storedUser = localStorage.getItem('user_profile')
    let parsedUser = null
    try {
        if (storedUser && storedUser !== 'undefined') {
            parsedUser = JSON.parse(storedUser)
        }
    } catch (e) {
        console.error('Failed to parse user profile from localStorage', e)
        localStorage.removeItem('user_profile')
    }
    const user = ref<any>(parsedUser)
    const loading = ref(false)

    // 2FA 挑战中间状态（不持久化到 localStorage）
    const challengeToken = ref<string>('')
    const challengeExpiresAt = ref<string>('')
    // 新环境登录邮箱验证挑战中间状态（不持久化）
    const emailVerifyChallengeToken = ref<string>('')
    const emailVerifyExpiresAt = ref<string>('')
    const emailVerifyMaskedEmail = ref<string>('')
    // 邀请制下第三方新用户的待补邀请码票据（不持久化，后端 10 分钟有效）
    const inviteTicket = ref<string>('')

    const isAuthenticated = computed(() => !!token.value)

    const setToken = (newToken: string) => {
        token.value = newToken
        localStorage.setItem('user_token', newToken)
    }

    const setUser = (newUser: any) => {
        user.value = newUser
        localStorage.setItem('user_profile', JSON.stringify(newUser))
    }

    const clearAuth = () => {
        token.value = ''
        user.value = null
        localStorage.removeItem('user_token')
        localStorage.removeItem('user_profile')
        resetApiAccountLinkCache()
    }

    const sendVerifyCode = async (payload: any) => {
        loading.value = true
        try {
            await userAuthAPI.sendVerifyCode(payload)
            return true
        } finally {
            loading.value = false
        }
    }

    const register = async (payload: any) => {
        loading.value = true
        try {
            const response = await userAuthAPI.register(payload)
            const { token: accessToken, user: userData } = response.data.data
            if (accessToken) {
                setToken(accessToken)
            }
            if (userData) {
                setUser(userData)
            }
            if (accessToken) {
                triggerSilentApiAccountLinkEnsure({ force: true })
            }
            return true
        } finally {
            loading.value = false
        }
    }

    const clearChallenge = () => {
        challengeToken.value = ''
        challengeExpiresAt.value = ''
    }

    const clearEmailVerifyChallenge = () => {
        emailVerifyChallengeToken.value = ''
        emailVerifyExpiresAt.value = ''
        emailVerifyMaskedEmail.value = ''
    }

    const login = async (payload: any) => {
        loading.value = true
        try {
            const response = await userAuthAPI.login(payload)
            return handleLoginResponse(response.data.data)
        } finally {
            loading.value = false
        }
    }

    const verifyLoginEmail = async (code: string) => {
        if (!emailVerifyChallengeToken.value) {
            throw new Error('challenge_token_missing')
        }
        loading.value = true
        try {
            const response = await userAuthAPI.verifyLoginEmail({
                challenge_token: emailVerifyChallengeToken.value,
                code,
            })
            const data = response.data.data || {}
            const { token: accessToken, user: userData } = data
            setToken(accessToken)
            setUser(userData)
            clearEmailVerifyChallenge()
            if (accessToken) {
                triggerSilentApiAccountLinkEnsure({ force: true })
            }
            return true
        } finally {
            loading.value = false
        }
    }

    const resendLoginEmail = async () => {
        if (!emailVerifyChallengeToken.value) {
            throw new Error('challenge_token_missing')
        }
        await userAuthAPI.resendLoginEmail({
            challenge_token: emailVerifyChallengeToken.value,
        })
        return true
    }

    const verify2FA = async (payload: { code?: string; recovery_code?: string }) => {
        if (!challengeToken.value) {
            throw new Error('challenge_token_missing')
        }
        loading.value = true
        try {
            const response = await userAuthAPI.verify2FA({
                challenge_token: challengeToken.value,
                ...payload,
            })
            const data = response.data.data || {}
            const { token: accessToken, user: userData } = data
            setToken(accessToken)
            setUser(userData)
            clearChallenge()
            if (accessToken) {
                triggerSilentApiAccountLinkEnsure({ force: true })
            }
            return true
        } finally {
            loading.value = false
        }
    }

    const handleLoginResponse = (data: any): { requiresTotp: boolean; requiresEmailVerify?: boolean; inviteRequired?: boolean } => {
        if (data?.invite_required) {
            inviteTicket.value = data.invite_ticket || ''
            return { requiresTotp: false, inviteRequired: true }
        }
        if (data?.requires_totp) {
            clearEmailVerifyChallenge()
            challengeToken.value = data.challenge_token || ''
            challengeExpiresAt.value = data.challenge_expires_at || ''
            return { requiresTotp: true }
        }
        if (data?.requires_email_verify) {
            clearChallenge()
            emailVerifyChallengeToken.value = data.challenge_token || ''
            emailVerifyExpiresAt.value = data.challenge_expires_at || ''
            emailVerifyMaskedEmail.value = data.masked_email || ''
            return { requiresTotp: false, requiresEmailVerify: true }
        }
        clearChallenge()
        clearEmailVerifyChallenge()
        const { token: accessToken, user: userData } = data || {}
        setToken(accessToken)
        setUser(userData)
        if (accessToken) {
            triggerSilentApiAccountLinkEnsure({ force: true })
        }
        return { requiresTotp: false }
    }

    const telegramLogin = async (payload: any) => {
        loading.value = true
        try {
            const response = await userAuthAPI.telegramLogin(payload)
            return handleLoginResponse(response.data.data)
        } finally {
            loading.value = false
        }
    }

    const telegramOidcLogin = async (payload: { code: string; state: string }) => {
        loading.value = true
        try {
            const response = await userAuthAPI.telegramOidcCallback(payload)
            return handleLoginResponse(response.data.data)
        } finally {
            loading.value = false
        }
    }

    const telegramMiniAppLogin = async (initData: string) => {
        loading.value = true
        try {
            const response = await userAuthAPI.telegramMiniAppLogin({ init_data: initData })
            return handleLoginResponse(response.data.data)
        } finally {
            loading.value = false
        }
    }

    const googleLogin = async (credential: string) => {
        loading.value = true
        try {
            const response = await userAuthAPI.googleLogin({ credential })
            return handleLoginResponse(response.data.data)
        } finally {
            loading.value = false
        }
    }

    const googleRedirectLogin = async () => {
        loading.value = true
        try {
            const response = await userAuthAPI.googleRedirectExchange()
            return handleLoginResponse(response.data.data)
        } finally {
            loading.value = false
        }
    }

    const clearInviteTicket = () => {
        inviteTicket.value = ''
    }

    const completeInviteRegistration = async (inviteCode: string) => {
        loading.value = true
        try {
            const response = await userAuthAPI.completeInviteRegistration({
                invite_ticket: inviteTicket.value,
                invite_code: inviteCode,
            })
            clearInviteTicket()
            return handleLoginResponse(response.data.data)
        } finally {
            loading.value = false
        }
    }

    const forgotPassword = async (payload: any) => {
        loading.value = true
        try {
            await userAuthAPI.forgotPassword(payload)
            return true
        } finally {
            loading.value = false
        }
    }

    const syncUserProfile = (profile: {
        id?: number
        email?: string
        nickname?: string
        locale?: string
        email_verified_at?: string | null
        email_change_mode?: 'bind_only' | 'change_with_old_and_new'
        password_change_mode?: 'set_without_old' | 'change_with_old'
    }) => {
        const nextUser = { ...(user.value || {}), ...profile }
        setUser(nextUser)
    }

    let lastSessionCheckAt = 0
    let sessionCheckInflight: Promise<boolean> | null = null

    const verifySession = async (force = false): Promise<boolean> => {
        const currentToken = token.value
        if (!currentToken) return false
        const now = Date.now()
        if (!force && now - lastSessionCheckAt < 15_000) return true
        if (sessionCheckInflight) return sessionCheckInflight
        lastSessionCheckAt = now
        const base = import.meta.env.VITE_API_BASE_URL || ''
        sessionCheckInflight = fetch(`${base}/api/v1/me`, {
            method: 'GET',
            headers: {
                Accept: 'application/json',
                Authorization: `Bearer ${currentToken}`,
            },
            cache: 'no-store',
        })
            .then(async (res) => {
                if (res.status === 401) {
                    clearAuth()
                    return false
                }
                if (!res.ok) return true
                const body = await res.json().catch(() => null)
                if (body && body.status_code === 401) {
                    clearAuth()
                    return false
                }
                return true
            })
            .catch(() => true)
            .finally(() => {
                sessionCheckInflight = null
            })
        return sessionCheckInflight
    }

    const logout = async (redirect = '/auth/login') => {
        const currentToken = token.value
        if (currentToken) {
            // Best-effort server-side SLO (bumps TokenVersion & revokes linked new-api session).
            const base = import.meta.env.VITE_API_BASE_URL || ''
            await Promise.race([
                fetch(`${base}/api/v1/me/logout`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        Authorization: `Bearer ${currentToken}`,
                    },
                    body: '{}',
                    keepalive: true,
                }).catch(() => undefined),
                new Promise(resolve => setTimeout(resolve, 1200)),
            ])
        }
        clearAuth()
        router.push(redirect)
    }

    return {
        token,
        user,
        loading,
        challengeToken,
        challengeExpiresAt,
        emailVerifyChallengeToken,
        emailVerifyExpiresAt,
        emailVerifyMaskedEmail,
        inviteTicket,
        isAuthenticated,
        sendVerifyCode,
        register,
        login,
        verifyLoginEmail,
        resendLoginEmail,
        verify2FA,
        clearChallenge,
        clearEmailVerifyChallenge,
        telegramLogin,
        telegramOidcLogin,
        telegramMiniAppLogin,
        googleLogin,
        googleRedirectLogin,
        clearInviteTicket,
        completeInviteRegistration,
        forgotPassword,
        syncUserProfile,
        verifySession,
        clearAuth,
        logout,
    }
})
