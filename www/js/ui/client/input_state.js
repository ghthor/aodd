// TODO rewrite this module as go
define(["underscore",
        "github.com/ghthor/aodd/game",
], function(_, game) {
    var InputState = function(inputConn) {
        var inputState = this;
        inputState.movement = [];
        inputState.assail = null;

        var sendMovement = function(direction) {
            inputConn.sendMoveRequest(game.MR_MOVE, direction);
        };

        var sendMovementCancel = function(direction) {
            inputConn.sendMoveRequest(game.MR_MOVE_CANCEL, direction);
        };

        var sendAssail = function() {
            inputConn.sendUseRequest(game.UR_USE, "assail");
        };

        var sendAssailCancel = function() {
            inputConn.sendUseRequest(game.UR_USE_CANCEL, "assail");
        };

        inputState.movementDown = function(direction) {
            if (!_.include(inputState.movement, direction)) {
                inputState.movement.push(direction);
                sendMovement(direction);
            }
        };

        inputState.movementUp = function(direction) {
            if (!_.include(inputState.movement, direction)) {
                throw "keystate already up";
            }
            if (_.indexOf(inputState.movement, direction) === (inputState.movement.length - 1)) {
                inputState.movement.pop();
                sendMovementCancel(direction);
            } else {
                inputState.movement = _.reject(inputState.movement, function(directionDown) { return directionDown === direction; });
            }

            if (inputState.movement.length > 0) {
                sendMovement(_.last(inputState.movement));
            }
        };

        inputState.assailDown = function() {
            inputState.assail = "assail";
            sendAssail();
        };

        inputState.assailUp = function() {
            inputState.assail = null;
            sendAssailCancel();
        };

        return this;
    };

    return InputState;
});
