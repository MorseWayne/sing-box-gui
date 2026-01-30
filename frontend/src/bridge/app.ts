import * as App from '@wails/go/bridge/App'

export const ReloadAppConfig = async () => {
  const { flag, data } = await App.ReloadAppConfig()
  if (!flag) {
    throw data
  }
  return data
}

export const GetEnv = App.GetEnv
export const ExitApp = App.ExitApp
export const ShowMainWindow = App.ShowMainWindow
export const UpdateTrayAndMenus = App.UpdateTrayAndMenus
export const UpdateTray = App.UpdateTray
export const UpdateTrayMenus = App.UpdateTrayMenus
export const IsStartup = App.IsStartup

export const RestartApp = async () => {
  const { flag, data } = await App.RestartApp()
  if (!flag) {
    throw data
  }
  return data
}

export const GetInterfaces = async () => {
  const { flag, data } = await App.GetInterfaces()
  if (!flag) {
    throw data
  }
  return data.split('|').filter((v) => v)
}
