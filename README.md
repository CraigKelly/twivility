# Twvility README

This is the twvility.com project.

## Quick intro

We are using godep so, yes, we commit the vendor directory to git. This isn't
what we want, but godep seems to be the dominant vendoring solutions right now
(according to the State of Go Survey 2016). Once there is an official golang
package manager, we'll switch to that.

## Tools

To save dependencies: `godep save` and then commit to repo

We provide four helpful scripts:

* `./test` - properly runs `godep go test` and includes the CL parameters we
   want. Accepts parameters to pass to `godep go test` (they do *not* override
   the default parameters)
* `./cover` - uses `./test` to get source code coverage and then display the
   HTML report. Notice that we use the build tag "test" to exclude main.go from
   unit tests *and* from coverage.
* `./build` - build twivility. We include the build date/time for display on
   startup. Note that we do *not* do `go install`
* `./run` - Run the binary built by `./build`, but first source the file
   `./authed` which you must create (see below)

## ./authed

You will need to create the `./authed` script yourself. A starter script
looks like this:

````bash
# Find these values at:
# https://dev.twitter.com/oauth/overview/application-owner-access-tokens
export TWITTER_CONSUMER_KEY="todo"
export TWITTER_CONSUMER_SECRET="todo"
export TWITTER_ACCESS_TOKEN="todo"
export TWITTER_ACCESS_SECRET="todo"
````
