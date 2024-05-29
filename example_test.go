package errkind

import "fmt"

func ExampleBadRequest() {
	// supply a message
	{
		err := BadRequest("message for bad request")
		fmt.Printf("%v (%d)\n", err, StatusCode(err))
	}

	// don't supply a message
	{
		err := BadRequest()
		fmt.Printf("%v (%d)\n", err, StatusCode(err))
	}

	// Output:
	// message for bad request (400)
	// bad request (400)
}

func ExampleForbidden() {
	// supply a message
	{
		err := Forbidden("message for forbidden")
		fmt.Printf("%v (%d)\n", err, StatusCode(err))
	}

	// don't supply a message
	{
		err := Forbidden()
		fmt.Printf("%v (%d)\n", err, StatusCode(err))
	}

	// Output:
	// message for forbidden (403)
	// forbidden (403)
}

func ExampleNotImplemented() {
	// supply a message
	{
		err := NotImplemented("message for not implemented")
		fmt.Printf("%v (%d)\n", err, StatusCode(err))
	}

	// don't supply a message
	{
		err := NotImplemented()
		fmt.Printf("%v (%d)\n", err, StatusCode(err))
	}

	// Output:
	// message for not implemented caller="example_test.go:44" (501)
	// not implemented caller="example_test.go:50" (501)
}
