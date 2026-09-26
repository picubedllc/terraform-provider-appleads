# Preferred when campaign id is known:
terraform import appleads_ad_group.main 1234567890/9876543210

# Bare ad group id (provider resolves campaign via find):
terraform import appleads_ad_group.main 9876543210

# Maximize Conversions (MAX_CONVERSIONS): after campaign create, list Apple's
# auto-created Automated Ad Group, then import it with one of the formats above.
# There is no separate automated-ad-group resource type.
