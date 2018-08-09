import {Action, ActionTypes} from 'src/dashboards/actions/v2'
import {Dashboard} from 'src/types/v2'

type State = Dashboard[]

export default (state: State = [], action: Action): State => {
  switch (action.type) {
    case ActionTypes.LoadDashboards: {
      const {dashboards} = action.payload

      return [...dashboards]
    }

    case ActionTypes.DeleteDashboard: {
      const {dashboard} = action.payload

      return [...state.filter(d => d.id !== dashboard.id)]
    }
  }
  return state
}
