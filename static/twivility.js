// routines specific to twivility: we assume jQuery and lodash are available

// Helpful mixins for lodash
(function() {
    function prop(obj, propname) {
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

    function trim(s) {
        if (!s && s !== 0 && s !== false)
            return "";

        var ss = "" + s;
        if (!ss || !ss.length || ss.length < 1) {
            return "";
        }

        if (ss.trim) {
            return ss.trim();
        }
        else {
            return ss.replace(/^\s+|\s+$/gm, '');
        }
    }

    function matchAny(s, checks) {
        var toCheck = trim(s);
        if (s === "")
            return false;
        for (var i = 0; i < checks.length; ++i) {
            if (toCheck.match(checks[i])) {
                return true;
            }
        }
        return false;
    }

    var leftChecks = [
        /clinton/i,
        /kaine/i
    ];
    function isLefty(s) {
        return matchAny(s, leftChecks);
    }

    var rightChecks = [
        /trump/i,
        /pence/i
    ];
    function isRighty(s) {
        return matchAny(s, rightChecks);
    }

    _.mixin({
        'prop': prop,
        'trim': trim,
        'isLefty': isLefty,
        'isRighty': isRighty
    });
})();



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
                //TODO: graceful failure
            });
    }

    // Get all accts, get a all tweets per acct, and then call acctCallback
    // with acct, tweetlist
    function getAllAccts(acctCallback, finishedCallback) {
        var count = 0;

        return $.get("/accts")
            .done(function(data) {
                _.each(data, function(acct){
                    count += 1;
                    getSingleAcct(acct, function(tweets){
                        if (!!acctCallback) {
                            acctCallback(acct, tweets);
                        }
                        count -= 1;
                        if (count < 1 && !!finishedCallback) {
                            finishedCallback();
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
