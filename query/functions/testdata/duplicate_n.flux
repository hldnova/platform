from(db: "test")
	|> range(start:2018-05-22T19:53:26Z)
	|> duplicate(columns: ["_time", "host"], n: 2)
