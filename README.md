# Twvility README

This is the twvility.com project.

## Quick intro

We are using godep so, yes, we commit the vendor directory to git. This isn't
what we want, but godep seems to be the dominant vendoring solutions right now
(according to the State of Go Survey 2016). Once there is an official golang
package manager, we'll switch to that.

We supply the `./run` script to running the compiled executable. Note that
this requires you to create your source-able file named `.authed`. See below.

Build, test, and run the server component with `cd srv && ./run.sh`

Run the server component with `./authed srv/twivility`

## Newbie help

To save dependencies: `godep save` and then commit to repo

To test: `godep go test` (optionally end with `-v` and `-race` flags)

To build: `go build` or `go install`


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
