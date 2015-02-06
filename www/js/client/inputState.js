define(["underscore"], function(_) {

    var InputState = function(socket) {
        var inputState = this;
        inputState.movement = [];
        inputState.time = 0;

        inputState.sendMovement = function(time, direction) {
            socket.send("3::move=" + time + ":" + direction);
        };

        inputState.sendMovementCancel = function(time, direction) {
            socket.send("3::moveCancel=" + time + ":" + direction);
        };

        inputState.movementDown = function(direction) {
            if (!_.include(inputState.movement, direction)) {
                inputState.movement.push(direction);
                inputState.sendMovement(inputState.time, direction);
            }
        };

        inputState.movementUp = function(direction) {
            if (!_.include(inputState.movement, direction)) {
                throw "keystate already up";
            }
            if (_.indexOf(inputState.movement, direction) === (inputState.movement.length - 1)) {
                inputState.movement.pop();
                inputState.sendMovementCancel(inputState.time, direction);
            } else {
                inputState.movement = _.reject(inputState.movement, function(directionDown) { return directionDown === direction; });
            }

            if (inputState.movement.length > 0) {
                inputState.sendMovement(inputState.time, _.last(inputState.movement));
            }
        };

        inputState.update = function(time) {
            if (inputState.movement.length > 0) {
                inputState.sendMovement(time, _.last(inputState.movement));
            }
            inputState.time = time;
        };

        return this;
    };

    return InputState;
});
