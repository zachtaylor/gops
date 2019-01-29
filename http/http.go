package http

import "ztaylor.me/gops"

func getHeader(o gops.Out, k string) string {
	if all := o.Headers()[k]; len(all) > 0 {
		return all[0]
	}
	return ""
}
