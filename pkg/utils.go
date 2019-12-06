package pkg

func contains(obj string, list []string) bool{
	for _, v := range list{
		if v == obj{
			return true
		}
	}
	return false
}
