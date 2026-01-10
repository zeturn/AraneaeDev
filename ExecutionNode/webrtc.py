# -*- coding: utf-8 -*-
"""
ExecutionNode WebRTC service

Provides:
- Session management for Playwright-driven pages
- SDP answer generation using aiortc
- Video track by periodically capturing Playwright page screenshots
- DataChannel 'ctrl' handling to inject mouse/keyboard events
- Human-in-the-loop resume signal support

Notes:
- Requires packages: aiortc, av, playwright
- Ensure browsers installed: `playwright install chromium`
"""
import asyncio
import json
import time
import uuid
import io
from typing import Dict, Optional

from aiortc import RTCPeerConnection, RTCSessionDescription, MediaStreamTrack
from aiortc.contrib.media import MediaBlackhole
from av import VideoFrame
from PIL import Image
import numpy as np

from playwright.sync_api import sync_playwright, Page, Browser, BrowserContext


class PlaywrightVideoTrack(MediaStreamTrack):
    kind = "video"

    def __init__(self, page: Page, fps: int = 10):
        super().__init__()
        self.page = page
        self.fps = fps
        self._running = True
        self._last_ts = time.time()

    async def recv(self) -> VideoFrame:
        # target frame interval
        interval = 1.0 / max(self.fps, 1)
        now = time.time()
        await asyncio.sleep(max(0.0, interval - (now - self._last_ts)))
        self._last_ts = time.time()

        # Capture screenshot (PNG bytes), convert to VideoFrame using PIL + numpy
        # Playwright sync API used via thread executor (avoid blocking loop)
        loop = asyncio.get_event_loop()
        png_bytes: bytes = await loop.run_in_executor(None, lambda: self.page.screenshot(type="png"))

        try:
            img = Image.open(io.BytesIO(png_bytes)).convert('RGB')
            arr = np.array(img)
            frame = VideoFrame.from_ndarray(arr, format='rgb24')
            return frame
        except Exception:
            # Fallback: black frame
            black = VideoFrame(width=1280, height=720, format="rgb24")
            return black


class Session:
    def __init__(self, url: str, headless: bool = True):
        self.id = uuid.uuid4().hex[:12]
        self.url = url
        self.headless = headless
        self.playwright = None  # type: ignore
        self.browser: Optional[Browser] = None
        self.context: Optional[BrowserContext] = None
        self.page: Optional[Page] = None
        self.pc: Optional[RTCPeerConnection] = None
        self.blackhole = MediaBlackhole()
        self.resume_event = asyncio.Event()

    def start_browser(self):
        if self.browser:
            return
        pw = sync_playwright().start()
        self.playwright = pw
        browser = pw.chromium.launch(headless=self.headless)
        context = browser.new_context(viewport={"width": 1280, "height": 720})
        page = context.new_page()
        page.goto(self.url)
        self.browser = browser
        self.context = context
        self.page = page

    def close_browser(self):
        try:
            if self.context:
                self.context.close()
            if self.browser:
                self.browser.close()
        finally:
            if self.playwright:
                self.playwright.stop()

    async def create_answer(self, offer_sdp: str, offer_type: str = "offer") -> Dict[str, str]:
        # ensure browser/page started
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, self.start_browser)

        assert self.page is not None

        pc = RTCPeerConnection()
        self.pc = pc

        # Add video track from Playwright page
        video_track = PlaywrightVideoTrack(self.page, fps=10)
        pc.addTrack(video_track)

        # Handle incoming datachannel for controls
        @pc.on("datachannel")
        def on_datachannel(channel):
            if channel.label != "ctrl":
                return

            @channel.on("message")
            def on_message(message):
                # Expect JSON control messages
                try:
                    if isinstance(message, bytes):
                        message = message.decode("utf-8")
                    data = json.loads(message)
                    asyncio.ensure_future(self._handle_ctrl(data))
                except Exception:
                    pass

        # Set remote description
        offer = RTCSessionDescription(sdp=offer_sdp, type=offer_type)
        await pc.setRemoteDescription(offer)

        # Create and set local description
        answer = await pc.createAnswer()
        await pc.setLocalDescription(answer)

        return {"sdp": pc.localDescription.sdp, "type": pc.localDescription.type}

    async def _handle_ctrl(self, data: Dict):
        if not self.page:
            return
        t = data.get("type")
        if t == "mouse":
            await self._handle_mouse(data)
        elif t == "keyboard":
            await self._handle_keyboard(data)
        elif t == "resume":
            self.resume_event.set()

    async def _handle_mouse(self, data: Dict):
        action = data.get("action")
        x = data.get("x", 0)
        y = data.get("y", 0)
        button = data.get("button", "left")
        delta_x = data.get("deltaX", 0)
        delta_y = data.get("deltaY", 0)
        page = self.page
        if not page:
            return
        loop = asyncio.get_event_loop()
        if action == "move":
            await loop.run_in_executor(None, lambda: page.mouse.move(x, y))
        elif action == "down":
            await loop.run_in_executor(None, lambda: page.mouse.down(button=button))
        elif action == "up":
            await loop.run_in_executor(None, lambda: page.mouse.up(button=button))
        elif action == "click":
            await loop.run_in_executor(None, lambda: page.mouse.click(x, y, button=button))
        elif action == "wheel":
            await loop.run_in_executor(None, lambda: page.mouse.wheel(delta_x, delta_y))

    async def _handle_keyboard(self, data: Dict):
        action = data.get("action")
        key = data.get("key")
        page = self.page
        if not page or not key:
            return
        loop = asyncio.get_event_loop()
        if action == "down":
            await loop.run_in_executor(None, lambda: page.keyboard.down(key))
        elif action == "up":
            await loop.run_in_executor(None, lambda: page.keyboard.up(key))
        elif action == "press":
            await loop.run_in_executor(None, lambda: page.keyboard.press(key))


class SessionStore:
    def __init__(self):
        self._sessions: Dict[str, Session] = {}

    def create(self, url: str, headless: bool = True) -> Session:
        s = Session(url=url, headless=headless)
        self._sessions[s.id] = s
        return s

    def get(self, session_id: str) -> Optional[Session]:
        return self._sessions.get(session_id)

    def remove(self, session_id: str):
        s = self._sessions.pop(session_id, None)
        if s:
            s.close_browser()


store = SessionStore()


async def api_create_session(url: str, headless: bool = True) -> Dict[str, str]:
    s = store.create(url=url, headless=headless)
    # lazy start browser when answering SDP; or start immediately:
    loop = asyncio.get_event_loop()
    await loop.run_in_executor(None, s.start_browser)
    return {"session_id": s.id}


async def api_offer(session_id: str, offer_sdp: str, offer_type: str = "offer") -> Dict[str, str]:
    s = store.get(session_id)
    if not s:
        raise ValueError("Invalid session_id")
    return await s.create_answer(offer_sdp, offer_type)


async def api_resume(session_id: str) -> Dict[str, str]:
    s = store.get(session_id)
    if not s:
        raise ValueError("Invalid session_id")
    s.resume_event.set()
    return {"status": "resumed"}
