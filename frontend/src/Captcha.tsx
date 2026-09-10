import { useEffect, useRef, useState } from "react"

type CaptchaProps = {
    masterImage: string
    titleImage: string
    dx: number
    dy: number
    onSubmit: (email: string, password: string, x: number, username?: string) => void
    registerRef: React.RefObject<HTMLDivElement | null>
    userNameRef?: React.RefObject<HTMLDivElement | null>
    passwordRef: React.RefObject<HTMLDivElement | null>
    loc: string
}

export default function Captcha({
    masterImage,
    titleImage,
    dx,
    dy,
    onSubmit,
    registerRef,
    userNameRef,
    passwordRef,
    loc
}: CaptchaProps) {

    const boxRef = useRef<HTMLDivElement>(null)
    const masterRef = useRef<HTMLImageElement>(null)

    const [sliderX, setSliderX] = useState(0)
    const [dragging, setDragging] = useState(false)
    const [scale, setScale] = useState(1)
    const [tileWidth, setTileWidth] = useState(0)

    const startX = useRef(0)
    const startSliderX = useRef(0)

    const updateScale = () => {
        if (!masterRef.current)
            return

        const master = masterRef.current

        if (master.naturalWidth === 0)
            return

        setScale(master.clientWidth / master.naturalWidth)
    }

    const startDrag = (clientX: number) => {
        startX.current = clientX
        startSliderX.current = sliderX
        setDragging(true)
    }

    const moveDrag = (clientX: number) => {
        if (!dragging || !boxRef.current)
            return

        const rect = boxRef.current.getBoundingClientRect()

        const delta = clientX - startX.current

        const initialX = dx * scale
        const displayedTileWidth = tileWidth

        const maxX =
            rect.width - initialX - displayedTileWidth

        let newX = startSliderX.current + delta

        if (newX < 0)
            newX = 0

        if (newX > maxX)
            newX = maxX

        setSliderX(newX)
    }

    const stopDrag = () => {
        setDragging(false)
    }

    useEffect(() => {
        const mouseMove = (e: MouseEvent) => {
            moveDrag(e.clientX)
        }

        const mouseUp = () => {
            stopDrag()
        }

        const touchMove = (e: TouchEvent) => {
            moveDrag(e.touches[0].clientX)
        }

        const touchEnd = () => {
            stopDrag()
        }

        window.addEventListener("mousemove", mouseMove)
        window.addEventListener("mouseup", mouseUp)
        window.addEventListener("touchmove", touchMove)
        window.addEventListener("touchend", touchEnd)

        return () => {
            window.removeEventListener("mousemove", mouseMove)
            window.removeEventListener("mouseup", mouseUp)
            window.removeEventListener("touchmove", touchMove)
            window.removeEventListener("touchend", touchEnd)
        }
    }, [dragging, sliderX, scale, tileWidth, dx])

    useEffect(() => {
        updateScale()

        window.addEventListener("resize", updateScale)

        return () => {
            window.removeEventListener("resize", updateScale)
        }
    }, [masterImage])

    return (
        <div className="captcha-box">

            <div
                className="captcha-image"
                ref={boxRef}
            >

                <img
                    ref={masterRef}
                    className="captcha-master"
                    src={masterImage}
                    onLoad={(e) => {
                        const master = e.currentTarget

                        const tile = new Image()

                        tile.onload = () => {
                            const scale =
                                master.clientWidth /
                                master.naturalWidth

                            setScale(scale)
                            setTileWidth(tile.naturalWidth * scale)
                        }

                        tile.src = titleImage
                    }}
                />

                <img
                    className="captcha-tile"
                    src={titleImage}
                    draggable={false}
                    style={{
                        position: "absolute",
                        left: `${dx * scale + sliderX}px`,
                        top: `${dy * scale}px`,
                        width: `${tileWidth}px`,
                    }}
                />

            </div>

            <div className="captcha-slider">

                <div className="captcha-slider-track">

                    <div
                        className="captcha-slider-button"
                        style={{
                            transform: `translateX(${sliderX}px)`
                        }}
                        onMouseDown={(e) => {
                            startDrag(e.clientX)
                        }}
                        onTouchStart={(e) => {
                            startDrag(e.touches[0].clientX)
                        }}
                    >
                        →
                    </div>

                </div>

            </div>

            <button
                className="captcha-submit"
                onClick={() => {

                    const finalX = Math.round((dx + sliderX) / 10) * 10
                    console.log("finalX our answer: " + finalX)
                    if (loc.includes("register"))
                        onSubmit(
                            registerRef.current!.innerText,
                            passwordRef.current!.innerText,
                            finalX,
                            userNameRef!.current!.innerText,
                        )
                    else if (loc.includes("login"))
                        onSubmit(
                            registerRef.current!.innerText,
                            passwordRef.current!.innerText,
                            finalX
                        )
                }}
            >
                Verify
            </button>

        </div>
    )
}