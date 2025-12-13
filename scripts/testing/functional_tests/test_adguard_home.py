import pytest
from playwright.sync_api import Page, expect
from testing import runners

@runners.tag('network', 'adguard-home', 'playwright')
def test_adguard_home_ui_loads(page: Page, config):
    """
    Checks that the AdGuard Home UI loads successfully in the browser.
    """
    adguard_home_url = config['urls'].get('adguard-home')
    assert adguard_home_url, "AdGuard Home URL not found in configuration."

    page.goto(adguard_home_url)
    
    # Expect a title "to contain" a substring.
    expect(page).to_have_title("AdGuard Home")
    
    # Expect a specific text content in the body, indicating the UI is loaded.
    expect(page.locator("body")).to_contain_text("Welcome")
    # or perhaps something more specific like a login form element
    # expect(page.locator('text=Install AdGuard Home')).to_be_visible() # or a login button

    print(f"  -> AdGuard Home UI loaded successfully at {adguard_home_url}")