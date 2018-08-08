import {QueryConfig} from 'src/types'
import {ColorString} from 'src/types/colors'

export interface Axis {
  label: string
  prefix: string
  suffix: string
  base: string
  scale: string
  bounds?: [string, string]
}

export type TimeSeriesValue = string | number | null | undefined

export interface FieldOption {
  internalName: string
  displayName: string
  visible: boolean
}

export interface TableOptions {
  verticalTimeAxis: boolean
  sortBy: FieldOption
  wrapping?: string
  fixFirstColumn: boolean
}

export interface Sort {
  field: string
  direction: string
}

export interface Axes {
  x: Axis
  y: Axis
  y2?: Axis
}

interface CellLinks {
  self: string
}

// corresponds to DashboardQuery on the backend
export interface CellQuery {
  query: string
  queryConfig: QueryConfig
  source: string
  text?: string // doesn't come from server
}

export interface Legend {
  type?: string
  orientation?: string
}

export interface DecimalPlaces {
  isEnforced: boolean
  digits: number
}

export interface Cell {
  id: string
  name: string
  visualization: V1Visualization | {}
}

export interface V1Visualization {
  type: string
  queries: CellQuery[]
  visualizationType: VisualizationType
  axes: Axes
  colors: ColorString[]
  tableOptions: TableOptions
  fieldOptions: FieldOption[]
  timeFormat: string
  decimalPlaces: DecimalPlaces
  links: CellLinks
  legend: Legend
  isWidget?: boolean
  inView: boolean
}

export enum VisualizationType {
  Line = 'line',
  Stacked = 'line-stacked',
  StepPlot = 'line-stepplot',
  Bar = 'bar',
  LinePlusSingleStat = 'line-plus-single-stat',
  SingleStat = 'single-stat',
  Gauge = 'gauge',
  Table = 'table',
  Alerts = 'alerts',
  News = 'news',
  Guide = 'guide',
}

interface DashboardLinks {
  self: string
}

export interface Dashboard {
  id: string
  cells: DashboardCell[]
  name: string
  links: DashboardLinks
}

export interface DashboardCell {
  x: number
  y: number
  w: number
  h: number
  ref: string
}

export enum ThresholdType {
  Text = 'text',
  BG = 'background',
  Base = 'base',
}
