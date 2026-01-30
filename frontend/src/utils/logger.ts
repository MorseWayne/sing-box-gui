import { AppendFile } from '@/bridge'
import { CoreWorkingDirectory } from '@/constant/kernel'

const AppLogFile = CoreWorkingDirectory + '/app.log'

class Logger {
    async log(message: string) {
        console.log(message)
        const time = new Date().toLocaleString()
        const content = `[${time}] [INFO] ${message}\n`
        await AppendFile(AppLogFile, content).catch(() => { })
    }

    async error(message: string) {
        console.error(message)
        const time = new Date().toLocaleString()
        const content = `[${time}] [ERROR] ${message}\n`
        await AppendFile(AppLogFile, content).catch(() => { })
    }
}

export const logger = new Logger()
