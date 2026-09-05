import { useEffect, useRef } from "react"
import { Link } from "react-router-dom"


export default function Header() {
    const searchBarRef = useRef<HTMLDivElement>(null)
    useEffect(() => {
        const v = searchBarRef.current
        if (v) {
            if (v.textContent == "") {
                v.textContent = "Search a post..."
                v.style.color = "gray"
            }

            v.addEventListener("focusin", () => {
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
            <div className="head-left">
                <div className="log">
                    Shayegan8
                </div>

                <div
                    className="searchbar"
                    contentEditable="true"
                    ref={searchBarRef}
                    data-placeholder="Search..."
                />
            </div>

            <div className="head-right">
                <Link className="posts" to="/posts">
                    Posts
                </Link>

                <div className="auth-links">
                    <Link className="auth" to="/login">Signin</Link>
                    <span>/</span>
                    <Link className="auth" to="/register">Signup</Link>
                </div>
            </div>
        </div>
    )
}