from(db:"testdb")
  |> range(start: 2018-05-22T19:53:26Z)
  |> filter(fn: (r) => r._measurement ==  "system" AND r._field == "load1")
  |> group(by: ["_measurement", "_start"])
  |> map(fn: (r) => {_time: r._time, load1:r._value})
  |> yield(name:"0")
