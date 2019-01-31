package main

import (
	"ztaylor.me/gops"
)

var Plugin = gops.New(
	gops.RouterFunc(router),
	gops.HandlerFunc(handler),
)

func router(i gops.In) bool {
	return true
}

func handler(i gops.In, o gops.Out) {
	o.Header("Content-Type", "text/html; charset=utf-8")
	o.StatusCode(500)
	o.Write(text)
}

func main() {
}

var text = []byte(`<html>
	<head>
		<title>GoPS - host not found!</title>
		<style type="text/css">
			body{
				font-size:36px;
				line-height:48px;
			}
		</style>
	</head>
	<body>
		Hello,<br/>
		<br/>
		This is Zach<br/>
		<br/>
		If your seeing this, you should probably just return to safety... <a href="http://ztaylor.me/">click here to go to my homepage</a><br/>
		<br/>
		...<br/>
		<br/>
		For technical info, please visit me on Github <a href="http://github.com/zachtaylor">github.com/zachtaylor</a><br/>
		<br/>
		<i>Powered by <a href="http://github.com/zachtaylor/gops">GoPS</a><i>
	</body>
</html>
`)
