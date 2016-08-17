// routines specific to twivility: we assume jQuery and lodash are available

// Helpful mixins for lodash
_.mixin({
    'prop': function(obj, propname) {
        if (_.isArray(obj) && _.isNumber(propname)) {
            return obj[propname];
        }
        else if ((!obj && obj !== "") || !propname || !_.has(obj, propname)) {
            return null;
        }
        else {
            return obj[propname];
        }
    }
});


// Actual twivility work
(function(t){
    // Get all tweets and call callback with them. Return the Future used for
    // the server query
    function getSingleAcct(acct, callback) {
        $.get("/tweets/" + acct)
            .done(function(data) {
                if (!!callback) {
                    callback(data);
                }
            })
            .fail(function(e) {
                console.log("GET Acct", acct, "FAILED:", e);
            });
    }

    // Get all accts, get a all tweets per acct, and then call acctCallback
    // with acct, tweetlist
    function getAllAccts(acctCallback) {
        return $.get("/accts")
            .done(function(data) {
                _.each(data, function(acct){
                    getSingleAcct(acct, function(tweets){
                        if (!!acctCallback) {
                            acctCallback(acct, tweets);
                        }
                    });
                });
            })
            .fail(function(e) {
                console.log("GET Accts FAILED:", e);
                //TODO: graceful failure
            });
    }

    t.GetSingleAcct = getSingleAcct;
    t.GetAllAccts = getAllAccts;
})(this);
