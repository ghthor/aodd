// TODO This module is 100% unspecified
define(["underscore",
       "client/bar",
       "client/sprite/human",
       "CAAT"
], function(_, Bar, Human) {

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

            var healthBar = new Bar(width, height/4, "red");
            healthBar.actor.
                setPositionAnchored(width/2, -height/2+4, 0.5, 0.5);
            actor.addChild(healthBar.actor);

            player.setHealthPercentage = function(percent) {
                healthBar.setPercent(percent);
            };

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
                if (!_.isNull(update.pathAction)) {
                    var pathAction = update.pathAction;

                    if (pathAction.start === time) {
                        var duration = pathAction.end - pathAction.start;
                        world.move(pathAction.orig, pathAction.dest, duration);
                    }
                }
                player.entity = update;
                actor.setAnimation(update);

                // update health display
                player.setHealthPercentage(update.hp/update.hpMax);
            };
            player.update(time, update);
        };

        return this;
    };

    return Player;
});
