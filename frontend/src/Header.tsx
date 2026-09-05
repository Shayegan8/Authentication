import { useEffect, useRef } from "react"


export default function Header() {
    const searchBarRef = useRef<HTMLDivElement>(null)
    useEffect(() => {
        const v = searchBarRef.current
        if (v) {
            if (v.textContent == "") {
                v.textContent = "Search a post..."
                v.style.color = "gray"
            }

            v.addEventListener("focus", () => {
                v.textContent = ""
                v.style.color = "black"
            })


            v.addEventListener("focusout", () => {
                if (v.textContent == "") {
                    v.textContent = "Search a post..."
                    v.style.color = "gray"
                }
            })

            v.addEventListener("keydown", (k) => {
                if (k.key == "Enter")
                    k.preventDefault()
            })
        }
    })

    return (
        <div className="head">
            <div>
                <div className="log">
                    Shayegan8
                </div>
                <div className="searchbar" contentEditable="true" ref={searchBarRef}>

                </div>
            </div>
            <div className="Posts">

            </div>
            <div className="auth"></div>
        </div>
    )
}