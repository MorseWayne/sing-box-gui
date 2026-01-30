import * as App from '@wails/go/bridge/App'

export interface CoreExecOptions {
    PidFile?: string
    Env?: Record<string, any>
    LogFile?: string
    StopOutputKeyword?: string
}

const mergeExecOptions = (options: CoreExecOptions) => {
    return {
        PidFile: options.PidFile ?? '',
        Convert: false,
        Env: options.Env ?? {},
        StopOutputKeyword: options.StopOutputKeyword ?? '',
        LogFile: options.LogFile ?? '',
    }
}

export const StartCore = async (path: string, args: string[], options: CoreExecOptions = {}) => {
    const { flag, data } = await App.StartCore(path, args, mergeExecOptions(options))
    if (!flag) {
        throw data
    }
    return Number(data)
}

export const StopCore = async () => {
    const { flag, data } = await App.StopCore()
    if (!flag) {
        throw data
    }
    return data
}
