define(["underscore",
        "CAAT",
], function(_) {
    var Bubble = function(width, height) {
        var bubble = this;

        bubble.actor = new CAAT.ActorContainer().setSize(width, height);

        // id of the say entity being displayed
        var entityId;

        // container for the sprite
        var chatBubble;

        bubble.setMsg = function(id, msg) {
            if (!_.isUndefined(chatBubble)) {
                bubble.clearMsg(entityId);
            }

            entityId = id;

            chatBubble = new CAAT.ActorContainer().setSize(width, height).
                setPositionAnchored(0, 0, 0, 0);

            bubble.actor.addChild(chatBubble);

            var divStyle = "'" + [
                "width: 100%",
                "height: 100%",

                "display: flex",
                "flex-direction: column",
                "justify-content: flex-end",
                "align-items: center",

                "font-size: 9pt",
            ].join(";") + ";'";

            var spanStyle = "'" + [
                "display: flex",
                "position: relative",
                "z-index: 1",

                "margin: 0",
                "padding: 5px",

                "border-style: solid",
                "border-width: 1px",
                "border-radius: 5px",

                "color: white",
            ].join(";") + ";'";

            var alphaStyle = "'" + [
                "position: absolute",
                "z-index: -1",

                "top: 0",
                "left: 0",
                "width: 100%",
                "height: 100%",

                "border-radius: 5px",

                "background-color: black",
                "opacity: 0.4",
            ].join(";") + ";'";

            var str = "<svg xmlns='http://www.w3.org/2000/svg' width='" + width + "' height='" + height + "'>" +
                "<foreignObject width='100%' height='100%' >" +
                    "<div xmlns='http://www.w3.org/1999/xhtml' style=" + divStyle + ">" +
                            "<span style=" + spanStyle + ">" +
                                 msg +
                                 "<div style=" + alphaStyle + "></div>" +
                            "</span>" +
                        "</div>" +
                    "</foreignObject>" +
                "</svg>";

            var DOMURL = window.URL || window.webkitURL || window;

            var img = new Image();
            var svg = new Blob([str], {type: "image/svg+xml;charset=utf-8"});
            var url = DOMURL.createObjectURL(svg);

            img.onload = function() {
                var spriteImage = new CAAT.SpriteImage().initialize(img, 1, 1);
                var sprite = new CAAT.Actor().
                    setBackgroundImage(spriteImage.getRef(), true).
                    setPositionAnchored(width/2, height, 0.5, 1);
                chatBubble.addChild(sprite);

                DOMURL.revokeObjectURL(url);
            };

            img.src = url;
        };

        bubble.clearMsg = function(id) {
            if (id === entityId) {
                bubble.actor.removeChild(chatBubble);
            }
        };

        return this;
    };

    return Bubble;
});
