import AJAX from 'src/utils/ajax'
import {Dashboard} from 'src/types/v2'

// TODO(desa): what to do about getting dashboards from another v2 source
export const getDashboards = async (url: string): Promise<Dashboard[]> => {
  try {
    const {data} = await AJAX({
      url,
    })

    return data.dashboards
  } catch (error) {
    throw error
  }
}

export const createDashboard = async (
  url: string,
  dashboard: Partial<Dashboard>
): Promise<Dashboard> => {
  try {
    const {data} = await AJAX({
      method: 'POST',
      url,
      data: dashboard,
    })

    return data
  } catch (error) {
    console.error(error)
    throw error
  }
}
