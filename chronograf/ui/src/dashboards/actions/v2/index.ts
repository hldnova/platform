// Types
import {Dispatch} from 'redux'
import {Dashboard} from 'src/types/v2'

// APIs
import {
  getDashboards as getDashboardsAJAX,
  createDashboard as createDashboardAJAX,
  deleteDashboard as deleteDashboardAJAX,
} from 'src/dashboards/apis/v2'

// Actions
import {notify} from 'src/shared/actions/notifications'

// Copy
import * as copy from 'src/shared/copy/notifications'

export enum ActionTypes {
  LoadDashboards = 'LOAD_DASHBOARDS',
  DeleteDashboard = 'DELETE_DASHBOARD',
  DeleteDashboardFailed = 'DELETE_DASHBOARD_FAILED',
}

export type Action = LoadDashboardsAction | DeleteDashboardAction

interface LoadDashboardsAction {
  type: ActionTypes.LoadDashboards
  payload: {
    dashboards: Dashboard[]
  }
}

interface DeleteDashboardAction {
  type: ActionTypes.DeleteDashboard
  payload: {
    dashboard: Dashboard
  }
}

interface DeleteDashboardFailedAction {
  type: ActionTypes.DeleteDashboardFailed
  payload: {
    dashboard: Dashboard
  }
}

// Action Creators

export const loadDashboards = (
  dashboards: Dashboard[]
): LoadDashboardsAction => ({
  type: ActionTypes.LoadDashboards,
  payload: {
    dashboards,
  },
})

export const deleteDashboard = (
  dashboard: Dashboard
): DeleteDashboardAction => ({
  type: ActionTypes.DeleteDashboard,
  payload: {dashboard},
})

export const deleteDashboardFailed = (
  dashboard: Dashboard
): DeleteDashboardFailedAction => ({
  type: ActionTypes.DeleteDashboardFailed,
  payload: {dashboard},
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
    dispatch(notify(copy.dashboardImported(name)))
  } catch (error) {
    dispatch(
      notify(copy.dashboardImportFailed('', 'Could not upload dashboard'))
    )
    console.error(error)
  }
}

export const deleteDashboardAsync = (dashboard: Dashboard) => async (
  dispatch: Dispatch<Action>
): Promise<void> => {
  dispatch(deleteDashboard(dashboard))

  try {
    await deleteDashboardAJAX(dashboard.links.self)
    dispatch(notify(copy.dashboardDeleted(dashboard.name)))
  } catch (error) {
    dispatch(
      notify(copy.dashboardDeleteFailed(dashboard.name, error.data.message))
    )

    dispatch(deleteDashboardFailed(dashboard))
  }
}
