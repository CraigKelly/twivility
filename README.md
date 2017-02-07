# Twivility README

This is the twivility.com project.

***IMPORTANT!*** The UI is incomplete and does nothing. You will be
disappointed if you try this software right now. Work is proceeding - do not
despair!

Until then, there is no license file. The plan is to release the code under
the MIT license (or possibly GPL v3). If you need licensing before then, what
are you thinking using pre-pre-alpha software? If you're sure, open an Issue
here and I'll fix the situation.

## Quick intro

We are using godep so, yes, we commit the vendor directory to git. This isn't
what we want, but godep seems to be the dominant vendoring solutions right now
(according to the State of Go Survey 2016). Once there is an official golang
package manager, we'll switch to that.

## Tools

We manage dependencies with godep. See below for the helper scripts in the
`./scripts` directory.

You can also use the `Makefile` in this directory. There is a make target
corresponding to each script name. The default target is `build`, which is
dependent on `test` so running `make` should do the right thing. Note that
`cover` always re-runs unit tests.

We provide helpful scripts in the scripts directory.

* `test` - properly runs `godep go test` and includes the CL parameters we
   want. Accepts parameters to pass to `godep go test` (they do *not* override
   the default parameters)
* `cover` - uses `test` to get source code coverage and then display the
   HTML report. Notice that we use the build tag "test" to exclude main.go from
   unit tests *and* from coverage.
* `build` - build twivility. We include the build date/time for display on
   startup. Note that we do *not* do `go install`
* `run` - Run the binary built by `build`, but first source the file
   `authed` which you must create (see below)
* `update` - Parse Godeps.json and then update them. Requires Python3

## authed

You will need to create the `authed` script yourself. A starter script
looks like this:

````bash
# Find these values at:
# https://dev.twitter.com/oauth/overview/application-owner-access-tokens
export TWITTER_CONSUMER_KEY="todo"
export TWITTER_CONSUMER_SECRET="todo"
export TWITTER_ACCESS_TOKEN="todo"
export TWITTER_ACCESS_SECRET="todo"
````

## License

This code is licensed under the MIT license. Please see `LICENSE`.

