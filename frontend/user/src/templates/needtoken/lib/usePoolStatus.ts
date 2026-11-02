import { ref } from 'vue'
import { STATUS_URL } from '../config'
import { parsePoolStatus, type PoolStatusResponse, type PoolStatusView } from './poolStatus'

// 页面间共享一份，没配置胶水地址时不请求
const view = ref<PoolStatusView | null>(null)
let inflight: Promise<void> | null = null

const load = async (): Promise<void> => {
    try {
        const res = await fetch(STATUS_URL, { headers: { Accept: 'application/json' }, credentials: 'omit' })
        if (!res.ok) throw new Error(`status http ${res.status}`)
        view.value = parsePoolStatus((await res.json()) as PoolStatusResponse)
    } catch {
        view.value = null // 拿不到就按“待接入”展示，不猜
    }
}

export const usePoolStatus = () => ({
    view,
    ensureLoaded: () => {
        if (!STATUS_URL || view.value || inflight) return inflight ?? Promise.resolve()
        inflight = load().finally(() => { inflight = null })
        return inflight
    },
})
