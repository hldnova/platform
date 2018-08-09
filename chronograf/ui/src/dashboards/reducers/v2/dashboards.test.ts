// Reducer
import reducer from 'src/dashboards/reducers/v2/dashboards'

// Actions
import {loadDashboards, deleteDashboard} from 'src/dashboards/actions/v2/'

// Resources
import {dashboard} from 'src/dashboards/reducers/v2/resources'

describe('dashboards reducer', () => {
  it('can load the dashboards', () => {
    const expected = [dashboard]
    const actual = reducer([], loadDashboards(expected))

    expect(actual).toEqual(expected)
  })

  it('can delete a dashboard', () => {
    const d2 = {...dashboard, id: '2'}
    const state = [dashboard, d2]
    const expected = [dashboard]
    const actual = reducer(state, deleteDashboard(d2))

    expect(actual).toEqual(expected)
  })
})
