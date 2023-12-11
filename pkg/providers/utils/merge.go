package utils

// MergeMaps will iterate recursively in src map and copy the fields over to dest
func MergeMaps(src, dest map[string]interface{}) {
	for key, val := range src {
		//If destination map already has this key then recursively
		//call merge with src[key] and dest[key]
		if dest[key] != nil {
			switch v := val.(type) {
			case map[string]interface{}:
				destMap, ok := dest[key].(map[string]interface{})
				if !ok {
					destMap = make(map[string]interface{})
				}
				MergeMaps(v, destMap)
			default:
				dest[key] = src[key]
			}
		}
	}
}
