define(["underscore",
       "client/sprite/human",
       "CAAT"
], function(_, Human) {

    var Player = function(director, world, entity) {
        var player = this;
        player.entity = entity;

        player.createActor = function(name, posX, posY, width, height) {
            var actor = new CAAT.ActorContainer().
                setSize(width, height).
                setPositionAnchored(posX, posY, 0.5, 0.5);

            var animation = new Human(director).
                makeSprites(Human.makeImage(director, "female"), actor.width/2, actor.height/2);
            _.each(animation.sprites, function(sprite) {
                actor.addChild(sprite);
            });

            var nameActor = new CAAT.TextActor().
                setBounds(0, -height, width, height).
                setTextAlign("center").
                setText(name);
            actor.addChild(nameActor);

            actor.setAnimation = function(entity) { animation.setAnimation(entity); };
            return actor;
        };

        // Create the actor when processing the first recieved update
        player.update = function(time, update) {
            var posX = world.scene.width/2,
                posY = world.scene.height/2,
                // TODO figure out a better value to use here
                width  = world.grid,
                height = world.grid;

            var actor = player.actor = player.createActor(entity.name, posX, posY, width, height);
            world.scene.addChild(actor);

            player.update = function(time, update) {
                if (!_.isNull(update.pathActions)) {
                    var pathAction = update.pathActions[0];

                    if (pathAction.start === time) {
                        var duration = pathAction.end - pathAction.start;
                        world.move(pathAction.orig, pathAction.dest, duration);
                    }
                }
                player.entity = update;
                actor.setAnimation(update);
            };
            player.update(time, update);
        };

        return this;
    };

    return Player;
});
