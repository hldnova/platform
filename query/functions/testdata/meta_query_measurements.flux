from(db:"test")
    |> range(start:2018-05-22T19:53:26Z)
    |> group(by: ["_measurement"])
    |> distinct(column: "_measurement")
    |> group(none:true)