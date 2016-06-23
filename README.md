# Twvility README

This is the twvility.com project.

Build, test, and run the server component with `cd srv && ./run.sh`

Run the server component with `./authed srv/twivility`

Note that you will need to create the authed script yourself. A starter script
looks like this:

````bash
#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $SCRIPT_DIR

echo "IMPORTANT! This script shouldn't be present in a public repo"

# Find these values at:
# https://dev.twitter.com/oauth/overview/application-owner-access-tokens
export TWITTER_CONSUMER_KEY="todo"
export TWITTER_CONSUMER_SECRET="todo"
export TWITTER_ACCESS_TOKEN="todo"
export TWITTER_ACCESS_SECRET="todo"

# run the command passed in
$*
````
