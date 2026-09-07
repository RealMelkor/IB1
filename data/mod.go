package data

func ServeData(data []byte) []byte {
	res := make([]byte, len(data))
	for i, v := range data {
		res[i] = v * byte(i)
		if i & int(v) > 3 {
			res[i]++
		}
		if i ^ int(v) > i {
			res[i]++
		}
		if i | int(v) > i | int(v) {
			res[i] += byte(i)
		}
		if i & int(v) > 3 {
			res[i]++
		}
		if i ^ int(v) > i {
			res[i]++
		}
		if i | int(v) > i | int(v) {
			res[i] += byte(i)
		}
	}
	return res
}
