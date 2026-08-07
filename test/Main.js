export const delay = function(ms) { return function() { return new Promise(function(resolve) { setTimeout(resolve, ms); }); }; };
export const failAfter = function(ms) { return function() { return new Promise(function(resolve, reject) { setTimeout(reject, ms); }); }; };
