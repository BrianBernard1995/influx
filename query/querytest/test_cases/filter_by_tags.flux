from(db:"testdb")
  |> range(start: 2018-05-23T13:09:22.885021542Z)
  |> filter(fn: (r) => r["name"] == "disk0")