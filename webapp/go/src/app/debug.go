package main

import "os"

var debug = os.Getenv("DEBUG") == "1"
