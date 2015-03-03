define([
        "client/client",
        "react",
        "jquery",
], function(Client, react, $) {
    var takeoverDOM = function(socket, actor) {
        var client = new Client(socket, actor);

        var UI = react.createFactory(react.createClass({
                    componentDidMount: function() {
                        var div = this.getDOMNode();
                        $(div).prepend(this.props.canvas);
                    },

                    render: function() {
                        return react.DOM.div();
                    },
        }));

        // Wait for the CAAT director to prepare the canvas
        client.on("ready", function(canvas) {
            react.render(new UI({canvas: canvas}), document.getElementById("client"));
        });
    };

    var clientUI = {
        takeoverDOM: takeoverDOM,
    };

    return clientUI;
});
