import React, {PureComponent} from 'react'
import {InjectedRouter} from 'react-router'
import {connect} from 'react-redux'
import download from 'src/external/download'
import _ from 'lodash'

import DashboardsContents from 'src/dashboards/components/DashboardsPageContents'
import PageHeader from 'src/reusable_ui/components/page_layout/PageHeader'
import {getDeep} from 'src/utils/wrappers'

import {createDashboard} from 'src/dashboards/apis/v2'
import {
  getDashboardsAsync,
  importDashboardAsync,
  deleteDashboardAsync,
} from 'src/dashboards/actions/v2'

import {retainRangesDashTimeV1 as retainRangesDashTimeV1Action} from 'src/dashboards/actions'
import {notify as notifyAction} from 'src/shared/actions/notifications'

import {
  NEW_DASHBOARD,
  DEFAULT_DASHBOARD_NAME,
  NEW_DEFAULT_DASHBOARD_CELL,
} from 'src/dashboards/constants'
import {ErrorHandling} from 'src/shared/decorators/errors'
import {
  dashboardExported,
  dashboardExportFailed,
  dashboardCreateFailed,
} from 'src/shared/copy/notifications'

import {Notification} from 'src/types/notifications'
import {DashboardFile, DashboardCell} from 'src/types/v2/dashboards'
import {Source, Links, Dashboard} from 'src/types/v2'

interface Props {
  source: Source
  router: InjectedRouter
  links: Links
  handleGetDashboards: typeof getDashboardsAsync
  handleDeleteDashboard: typeof deleteDashboardAsync
  handleImportDashboard: typeof importDashboardAsync
  notify: (message: Notification) => void
  retainRangesDashTimeV1: (dashboardIDs: string[]) => void
  dashboards: Dashboard[]
}

@ErrorHandling
class DashboardsPage extends PureComponent<Props> {
  public async componentDidMount() {
    const {handleGetDashboards, dashboards, links} = this.props
    await handleGetDashboards(links.dashboards)
    const dashboardIDs = dashboards.map(d => d.id)
    this.props.retainRangesDashTimeV1(dashboardIDs)
  }

  public render() {
    const {dashboards, notify} = this.props
    const dashboardLink = `/sources/${this.props.source.id}`

    return (
      <div className="page">
        <PageHeader titleText="Dashboards" sourceIndicator={true} />
        <DashboardsContents
          dashboardLink={dashboardLink}
          dashboards={dashboards}
          onDeleteDashboard={this.handleDeleteDashboard}
          onCreateDashboard={this.handleCreateDashboard}
          onCloneDashboard={this.handleCloneDashboard}
          onExportDashboard={this.handleExportDashboard}
          onImportDashboard={this.handleImportDashboard}
          notify={notify}
        />
      </div>
    )
  }

  private handleCreateDashboard = async (): Promise<void> => {
    const {source, links, router, notify} = this.props
    try {
      const data = await createDashboard(links.dashboards, NEW_DASHBOARD)
      router.push(`/sources/${source.id}/dashboards/${data.id}`)
    } catch (error) {
      notify(dashboardCreateFailed())
    }
  }

  private handleCloneDashboard = (dashboard: Dashboard) => async (): Promise<
    void
  > => {
    const {source, router, links, notify} = this.props
    const name = `${dashboard.name} (clone)`
    try {
      const data = await createDashboard(links.dashboards, {
        ...dashboard,
        name,
      })
      router.push(`/sources/${source.id}/dashboards/${data.id}`)
    } catch (error) {
      notify(dashboardCreateFailed())
    }
  }

  private handleDeleteDashboard = (dashboard: Dashboard) => (): void => {
    this.props.handleDeleteDashboard(dashboard)
  }

  private handleExportDashboard = (dashboard: Dashboard) => async (): Promise<
    void
  > => {
    const dashboardForDownload = await this.modifyDashboardForDownload(
      dashboard
    )
    try {
      download(
        JSON.stringify(dashboardForDownload, null, '\t'),
        `${dashboard.name}.json`,
        'text/plain'
      )
      this.props.notify(dashboardExported(dashboard.name))
    } catch (error) {
      this.props.notify(dashboardExportFailed(dashboard.name, error))
    }
  }

  private modifyDashboardForDownload = async (
    dashboard: Dashboard
  ): Promise<DashboardFile> => {
    return {meta: {chronografVersion: '2.0'}, dashboard}
  }

  private handleImportDashboard = async (
    dashboard: Dashboard
  ): Promise<void> => {
    const {links} = this.props
    const name = _.get(dashboard, 'name', DEFAULT_DASHBOARD_NAME)
    const cellsWithDefaultsApplied = getDeep<DashboardCell[]>(
      dashboard,
      'cells',
      []
    ).map(c => ({...NEW_DEFAULT_DASHBOARD_CELL, ...c}))

    await this.props.handleImportDashboard(links.dashboards, {
      ...dashboard,
      name,
      cells: cellsWithDefaultsApplied,
    })
  }
}

const mstp = state => {
  const {dashboards, links} = state

  return {
    dashboards,
    links,
  }
}

const mdtp = {
  notify: notifyAction,
  handleGetDashboards: getDashboardsAsync,
  handleDeleteDashboard: deleteDashboardAsync,
  handleImportDashboard: importDashboardAsync,
  retainRangesDashTimeV1: retainRangesDashTimeV1Action,
}

export default connect(mstp, mdtp)(DashboardsPage)
