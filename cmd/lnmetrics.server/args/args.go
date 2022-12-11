package args

var CliArgs struct {
	Port   string `default:"8080" help:"Set the server port where the app will run."`
	DbPath string `help:"Set the database root path where the server database are build"`
}
