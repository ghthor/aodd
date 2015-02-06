define(["client/inputState",
       "underscore",
       "lib/jasmine"
], function(InputState, _) {

    var MockSocket = function() {};

    describe("input state manager", function() {
        var mockSocket, inputState;

        beforeEach(function() {
            mockSocket = new MockSocket();
            inputState = new InputState(mockSocket);

            mockSocket.send = jasmine.createSpy("socket.send");
        });

        describe("managing movement keys", function() {
            var directions;
            beforeEach(function() {
                directions = ["north", "south", "east", "west"];
                _.each(directions, function(direction) {
                    inputState.movementDown(direction);
                });
            });

            it("mantains a stack of movement keys that are held down", function() {
                directions.reverse();
                _.each(directions, function(direction) {
                    expect(inputState.movement.pop()).toBe(direction);
                });
            });

            it("if a movement key is released it is removed from the stack", function() {
                var directions = ["south", "north", "west", "east"];
                _.each(directions, function(direction) {
                    inputState.movementUp(direction);
                    expect(inputState.movement).not.toContain(direction);
                });
            });

            it("location in stack doesn't change if down is triggered again", function() {
                directions.reverse();
                _.each(directions, function(direction) {
                    var length = inputState.movement.length,
                        index = _.indexOf(inputState.movement, direction);

                    inputState.movementDown(direction);
                    expect(inputState.movement.length).toBe(length);
                    expect(_.indexOf(inputState.movement, direction)).toBe(index);
                });
            });

            it("throws an error if up is triggered twice for a direction", function() {
                var directions = ["north", "south", "east", "west"];
                _.each(directions, function(direction) {
                    expect(function() { inputState.movementUp(direction); }).not.toThrow();
                    expect(function() { inputState.movementUp(direction); }).toThrow("keystate already up");
                });
            });
        });

        describe("movement input events are sent", function() {
            it("when a movement key is added to the top of the stack", function() {
                var directions = ["north", "east", "south", "west"];
                _.each(directions, function(direction, i) {
                    inputState.movementDown(direction);
                    expect(mockSocket.send.calls.length).toBe(i + 1);
                    expect(mockSocket.send.calls[i].args[0]).toBe("3::move=0:" + direction);
                });
            });

            it("when the client recieves an update from the server", function() {
                spyOn(inputState, "sendMovement").andCallThrough();

                var time = 0;
                inputState.movement = ["north"];

                time++;
                inputState.update(time);

                expect(inputState.sendMovement).toHaveBeenCalled();
                expect(mockSocket.send.calls[0].args[0]).toBe("3::move=1:north");

                inputState.movementUp("north");

                time++;
                inputState.update(time);

                expect(inputState.sendMovement.calls.length).toBe(1);
            });

            describe("when movement keys are released", function() {
                it("a move cancel event is sent", function() {
                    inputState.movement = ["north"];
                    inputState.movementUp("north");
                    expect(mockSocket.send).toHaveBeenCalled();
                    expect(mockSocket.send.calls.length).toBe(1);
                    expect(mockSocket.send.calls[0].args[0]).toBe("3::moveCancel=0:north");
                });

                it("a move event is sent if there is a remaining movement key pressed", function() {
                    spyOn(inputState, "sendMovementCancel");
                    spyOn(inputState, "sendMovement");
                    inputState.movement = ["south", "north"];
                    inputState.movementUp("north");

                    expect(inputState.sendMovementCancel).toHaveBeenCalled();
                    _.each([0, "north"], function(arg, i) {
                        expect(inputState.sendMovementCancel.calls[0].args[i]).toBe(arg);
                    });

                    expect(inputState.sendMovement).toHaveBeenCalled();
                    _.each([0, "south"], function(arg, i) {
                        expect(inputState.sendMovement.calls[0].args[i]).toBe(arg);
                    });
                });
            });
        });
    });
});
