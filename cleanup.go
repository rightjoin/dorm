package dorm

var processExiting = false

func Cleanup() {

	// Yes, we are quitting
	processExiting = true

	// TODO:
	// cleanup all subscriber processes
}
