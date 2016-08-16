angular.module('syncthing.core')
    .filter('jsonToHuman', function () {
        'use strict';

        return function (input) {
            var result = input.charAt(0).toUpperCase() + input.slice(1).replace(/([A-Z])/g, function (match) {
                return " " + match.toLowerCase();
            });
            var suffix = '';
            switch (input[input.length-1]) {
                case 'H':
                    suffix = '(in hours)';
                    break;
                case 'M':
                    suffix = '(in minutes)';
                    break;
                case 'S':
                    suffix = '(in seconds)';
                    break;
            }
            if (suffix) {
                result = result.slice(0, -1) + suffix;
            }
            if (result.slice(-3) == 'pct') {
                result = result.slice(0, -3) + '(in percent)';
            }
            return result;
        };
    });
