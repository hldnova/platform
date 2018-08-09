// Libraries
import React, {PureComponent} from 'react'
import {withRouter, Link, WithRouterProps} from 'react-router'
import {connect} from 'react-redux'
import _ from 'lodash'

// Components
import {NavBlock, NavHeader} from 'src/side_nav/components/NavItems'

// Constants
import {DEFAULT_HOME_PAGE} from 'src/shared/constants'

// Types
import {Source} from 'src/types/v2'

import {ErrorHandling} from 'src/shared/decorators/errors'

interface Props extends WithRouterProps {
  sources: Source[]
  isHidden: boolean
}

@ErrorHandling
class SideNav extends PureComponent<Props> {
  constructor(props) {
    super(props)
  }

  public render() {
    const {location, isHidden, sources = []} = this.props

    const {pathname, query} = location
    const defaultSource = sources.find(s => s.default)
    const id = query.sourceID || _.get(defaultSource, 'id', 0)
    console.log(sources)

    const sourceParam = `?sourceID=${id}`
    const isDefaultPage = pathname.split('/').includes(DEFAULT_HOME_PAGE)

    return isHidden ? null : (
      <nav className="sidebar">
        <div
          className={isDefaultPage ? 'sidebar--item active' : 'sidebar--item'}
        >
          <Link
            to={`/status/${sourceParam}`}
            className="sidebar--square sidebar--logo"
          >
            <span className="sidebar--icon icon cubo-uniform" />
          </Link>
        </div>
        <NavBlock
          highlightWhen={['delorean']}
          icon="capacitor2"
          link={`/delorean${sourceParam}`}
          location={pathname}
        >
          <NavHeader link={`/delorean/${sourceParam}`} title="Flux Editor" />
        </NavBlock>
        <NavBlock
          highlightWhen={['dashboards']}
          icon="dash-j"
          link="dashboards"
          location={pathname}
        >
          <NavHeader link="dashboards" title="Dashboards" />
        </NavBlock>
        <NavBlock
          highlightWhen={['logs']}
          icon="wood"
          link="/logs"
          location={pathname}
        >
          <NavHeader link={'/logs'} title="Log Viewer" />
        </NavBlock>
        <NavBlock
          highlightWhen={['manage-sources', 'kapacitors']}
          icon="wrench"
          link={`/manage-sources/${sourceParam}`}
          location={pathname}
        >
          <NavHeader
            link={`/manage-sources/${sourceParam}`}
            title="Configuration"
          />
        </NavBlock>
      </nav>
    )
  }
}

const mapStateToProps = ({
  sources,
  app: {
    ephemeral: {inPresentationMode},
  },
}) => ({
  sources,
  isHidden: inPresentationMode,
})

export default connect(mapStateToProps)(withRouter(SideNav))
