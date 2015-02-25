define(["CAAT",
], function() {
    var Bar = function(width, height, color) {
        var bar = this;

        bar.actor = new CAAT.ActorContainer().
            setSize(width, height);

        var rect = new CAAT.ShapeActor().
            setShape(CAAT.ShapeActor.SHAPE_RECTANGLE).
            setSize(width, height).
            setPositionAnchored(0, 0, 0, 0).
            setFillStyle(color);
        bar.actor.addChild(rect);

        bar.setPercent = function(percent) {
            rect.setSize(width * percent, height);
        };

        return this;
    };

    return Bar;
});
