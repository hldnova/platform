// Reducer
import reducer from 'src/dashboards/reducers/v2/dashboards'

// Actions
import {loadDashboards} from 'src/dashboards/actions/v2/'

// Resources
import {dashboard} from 'src/dashboards/reducers/v2/tests/resources'

describe('dashboards reducer', () => {
  it(`can load the dashboards`, () => {
    const expected = [dashboard]
    const actual = reducer([], loadDashboards(expected))

    expect(actual).toEqual(expected)
  })
})
