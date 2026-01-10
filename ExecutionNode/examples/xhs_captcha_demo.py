# -*- coding: utf-8 -*-
"""
Example: Xiaohongshu (小红书) captcha HITL demo using Playwright + WebRTC

Workflow:
1) Create a WebRTC session on ExecutionNode for a target URL (XHS explore page, for example)
2) Print session_id and an operator URL for ControlNode WebRTC page
3) If captcha appears (or always require intervention), wait for human to solve it
4) Operator clicks "Resume" on ControlNode page -> resume_event set -> continue automation

Run:
  python -m ExecutionNode.examples.xhs_captcha_demo

Note:
- Ensure ExecutionNode Flask app is running (for SDP offer/answer endpoints)
- Ensure ControlNode Django app is running (for webrtc/session page)
- Install browsers: `python -m playwright install chromium`
"""
import asyncio
import os

# Adjust these hosts if needed
CONTROL_HOST = os.environ.get('CONTROL_HOST', 'http://127.0.0.1:8000')
EXEC_HOST    = os.environ.get('EXEC_HOST',    'http://127.0.0.1:5001')
TARGET_URL   = os.environ.get('TARGET_URL',   'https://www.xiaohongshu.com/explore')


async def main():
    from ExecutionNode.webrtc import api_create_session, store

    # 1) Create session and start Playwright at target URL
    res = await api_create_session(url=TARGET_URL, headless=True)
    session_id = res['session_id']

    # 2) Print operator URL for ControlNode page (pre-fills exec host and session_id)
    operator_url = f"{CONTROL_HOST}/webrtc/session/?exec={EXEC_HOST}&session_id={session_id}"
    print("[HITL] Please open operator page to connect and solve captcha:")
    print(f"       {operator_url}")

    # 3) Obtain session and page, detect captcha DOM, wait for resume
    s = store.get(session_id)
    if not s:
        print("[ERROR] session not found in store")
        return

    # Use sync Playwright page from session
    page = s.page
    if not page:
        print("[ERROR] page not ready")
        return

    # Detect captcha presence; for demo, if #red-captcha exists or is visible, we wait
    try:
        page.wait_for_load_state('load')
    except Exception:
        pass

    try:
        has_captcha = page.locator('#red-captcha').count() > 0
    except Exception:
        has_captcha = False

    if has_captcha:
        print('[HITL] Captcha detected. Waiting for human to solve, then click Resume…')
        await s.resume_event.wait()
        print('[HITL] Resume signal received. Continue automation…')
    else:
        print('[HITL] No captcha found (or undetected). Proceeding…')

    # 4) Continue: simple example - print title and first 120 chars of content
    try:
        title = page.title()
        html  = page.content()
        print('[RESULT] Page title:', title)
        print('[RESULT] Content preview:', html[:120].replace('\n',' '), '...')
    except Exception as e:
        print('[ERROR] Continue automation failed:', e)


if __name__ == '__main__':
    asyncio.run(main())
