async def get_inventory(ctx, args):
    """Exemple d'outil pour Lisa"""
    door = await ctx.ha.get_state("binary_sensor.fridge_door")
    return f"Le frigo est {'ouvert' if door == 'on' else 'fermé'}."
