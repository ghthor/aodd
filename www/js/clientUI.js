define([
        "client/client",
        "react",
        "jquery",
], function(Client, react, $) {
    var takeoverDOM = function(socket, actor) {
        var client = new Client(socket, actor);

        // Wait for the CAAT director to prepare the canvas
        client.on("canvasReady", function(canvas) {
            react.render(react.DOM.div({id: "clientCanvas"}), document.body);
            $("#clientCanvas").append(canvas);
        });
    };

    var clientUI = {
        takeoverDOM: takeoverDOM,
    };

    return clientUI;
});
