// +build !test

/*
Twivility is a service that track a specific Twitter account's feeds, and
provides a web interface with some simple analysis.

Important! Any of the three commands require all four environment variables
to be set. See "Environment Variables".

Security note: all four environment variables have corresponding command line
flags (for instance, you can use `--consumer-key=yadda` instead of setting
the environment variable TWITTER_CONSUMER_KEY). HOWEVER, that is generally
a bad idea since the command line can be viewed with the `ps` command.

Usage:
    twivility [flags] cmd

Commands:
    service
        Run the Twivility service (by default serving at Orwellian port 8484).
        Includes the HTML client/site (served at "/"). The service will
        occasionally query Twitter for new tweets.

    update
        Updates the local store of stored tweets. Note that no synchronization
        will be attempted with a running copy of the service, so you should
        make sure to run this when no other instance of twivility is active.

    dump
        Dump all tweets stored to stdout as a JSON object.

    json
        A synonym for the "dump" command

Flags:
    -host <address binding string>
        The default value is "127.0.0.1:8484". Note that this flag only has an
        effect when using the "service" command

Environment Variables:
    TWITTER_CONSUMER_KEY
    TWITTER_CONSUMER_SECRET
    TWITTER_ACCESS_TOKEN
    TWITTER_ACCESS_SECRET

    These values can be found at
    https://dev.twitter.com/oauth/overview/application-owner-access-tokens
*/
package main
