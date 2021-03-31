server:
	go build && DOMAIN=localhost PORT=45001 ./aodd -heroku=false
