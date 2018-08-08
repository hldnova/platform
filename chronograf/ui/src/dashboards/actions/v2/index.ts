// Types
import {Dispatch} from 'redux'
import {Dashboard} from 'src/types/v2'

// APIs
import {
  getDashboards as getDashboardsAJAX,
  createDashboard as createDashboardAJAX,
} from 'src/dashboards/apis/v2'

// Actions
import {notify} from 'src/shared/actions/notifications'

// Copy
import {
  notifyDashboardImported,
  notifyDashboardImportFailed,
} from 'src/shared/copy/notifications'

export enum ActionTypes {
  LoadDashboards = 'LOAD_DASHBOARDS',
}

interface LoadDashboardsAction {
  type: ActionTypes.LoadDashboards
  payload: {
    dashboards: Dashboard[]
  }
}

export type Action = LoadDashboardsAction

// Action Creators

export const loadDashboards = (
  dashboards: Dashboard[]
): LoadDashboardsAction => ({
  type: ActionTypes.LoadDashboards,
  payload: {
    dashboards,
  },
})

// Thunks

export const getDashboardsAsync = (url: string) => async (
  dispatch: Dispatch<Action>
): Promise<Dashboard[]> => {
  try {
    const dashboards = await getDashboardsAJAX(url)
    dispatch(loadDashboards(dashboards))
    return dashboards
  } catch (error) {
    console.error(error)
    throw error
  }
}

export const importDashboardAsync = (
  url: string,
  dashboard: Dashboard
) => async (dispatch: Dispatch<Action>): Promise<void> => {
  try {
    await createDashboardAJAX(url, dashboard)
    const dashboards = await getDashboardsAJAX(url)

    dispatch(loadDashboards(dashboards))
    dispatch(notify(notifyDashboardImported(name)))
  } catch (error) {
    dispatch(
      notify(notifyDashboardImportFailed('', 'Could not upload dashboard'))
    )
    console.error(error)
  }
}
