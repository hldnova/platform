import {Dispatch} from 'redux'
import {Dashboard} from 'src/types/v2'
import {getDashboards as getDashboardsAJAX} from 'src/dashboards/apis/v2'

export enum ActionType {
  LoadDashboards = 'LOAD_DASHBOARDS',
}

interface LoadDashboardsAction {
  type: ActionType.LoadDashboards
  payload: {
    dashboards: Dashboard[]
  }
}

export type Action = LoadDashboardsAction

// Action Creators

export const loadDashboards = (
  dashboards: Dashboard[]
): LoadDashboardsAction => ({
  type: ActionType.LoadDashboards,
  payload: {
    dashboards,
  },
})

// Thunkers

export const getDashboardsAsync = (url: string) => async (
  dispatch: Dispatch<Action>
): Promise<Dashboard[]> => {
  try {
    const dashboards = await getDashboardsAJAX(url)
    dispatch(loadDashboards(dashboards))
    return dashboards
  } catch (error) {
    console.error(error)
  }
}
