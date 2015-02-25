define(["underscore"], function(_) {

    var InputState = function(socket) {
        var inputState = this;
        inputState.movement = [];
        inputState.assail = null;
        inputState.time = 0;

        var sendMovement = function(time, direction) {
            socket.send("3::move=" + time + ":" + direction);
        };

        var sendMovementCancel = function(time, direction) {
            socket.send("3::moveCancel=" + time + ":" + direction);
        };

        var sendAssail = function(time) {
            socket.send("3::use=" + time + ":" + "assail");
        };

        var sendAssailCancel = function(time) {
            socket.send("3::useCancel=" + time + ":" + "assail");
        };

        inputState.movementDown = function(direction) {
            if (!_.include(inputState.movement, direction)) {
                inputState.movement.push(direction);
                sendMovement(inputState.time, direction);
            }
        };

        inputState.movementUp = function(direction) {
            if (!_.include(inputState.movement, direction)) {
                throw "keystate already up";
            }
            if (_.indexOf(inputState.movement, direction) === (inputState.movement.length - 1)) {
                inputState.movement.pop();
                sendMovementCancel(inputState.time, direction);
            } else {
                inputState.movement = _.reject(inputState.movement, function(directionDown) { return directionDown === direction; });
            }

            if (inputState.movement.length > 0) {
                sendMovement(inputState.time, _.last(inputState.movement));
            }
        };

        inputState.assailDown = function() {
            inputState.assail = "assail";
            sendAssail(inputState.time);
        };

        inputState.assailUp = function() {
            inputState.assail = null;
            sendAssailCancel(inputState.time);
        };

        inputState.update = function(time) {
            if (inputState.movement.length > 0) {
                sendMovement(time, _.last(inputState.movement));
            }

            if (!_.isNull(inputState.assail)) {
                sendAssail(time);
            }

            inputState.time = time;
        };

        return this;
    };

    return InputState;
});
