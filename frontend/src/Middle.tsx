import { useContext, useEffect, useRef, useState } from "react"
import { useLocation, useNavigate, useParams } from "react-router-dom"
import { RootElementContext } from "./RootElementContext"
import Captcha from "./Captcha"
import { login, loginValidation, loginValidationJWT, loginValidationSubmit, register, registerValidation, registerValidationJWT, registerValidationSubmit } from "./api"

export default function Middle() {
    const navigate = useNavigate()
    const validationFucker = async (email: string, password: string, x: number, username?: string) => {
        let check
        let res
        if (username) {
            check = await registerValidation(email, username, password, x)
            res = await registerValidationJWT()
        } else {
            check = await loginValidation(email, password, x)
            res = await loginValidationJWT()
        }
        rooti!.removeChild(captchaBackground.current!)
        rooti!.removeChild(captchaMenu.current!)
        if (res)
            await navigate("/errorFuck")
        else
            if (!check) { // if it wasnt 202 
                setVerification(true)
                console.log("I SET THIS MOTHER FUCKER")
                if (username)
                    await navigate("/register/validation/jwt")
                else
                    await navigate("/login/validation/jwt")
            } else
                await navigate("/error")
    }
    const loc = useLocation()
    const params = useParams()
    const [verification, setVerification] = useState(false)
    const registerRef = useRef<HTMLDivElement>(null)
    const passwordRef = useRef<HTMLDivElement>(null)
    const userNameRef = useRef<HTMLDivElement>(null)
    const verificationRefs = useRef<Array<HTMLInputElement | null>>([])
    const submitRegister = useRef<HTMLDivElement>(null)
    let Elementa: React.JSX.Element
    const captchaBackground = useRef<HTMLDivElement>(null)
    const captchaMenu = useRef<HTMLDivElement>(null)
    const rooti = useContext(RootElementContext)
    const handleVerificationInput = (
        e: React.ChangeEvent<HTMLInputElement>,
        index: number
    ) => {
        const value = e.target.value

        // Keep only numbers
        const digit = value.replace(/\D/g, "").slice(-1)

        e.target.value = digit

        if (digit && index < 4) {
            verificationRefs.current[index + 1]?.focus()
        }
    }

    const handleVerificationKeyDown = (
        e: React.KeyboardEvent<HTMLInputElement>,
        index: number
    ) => {
        if (e.key === "Backspace")
            if (!e.currentTarget.value && index > 0)
                verificationRefs.current[index - 1]?.focus()

        if (!/^[0-9]$/.test(e.key) && e.key !== "Backspace" && e.key !== "Delete" &&
            e.key !== "ArrowLeft" &&
            e.key !== "ArrowRight" &&
            e.key !== "Tab")
            e.preventDefault()
    }

    const handleVerificationPaste = (
        e: React.ClipboardEvent<HTMLInputElement>
    ) => {
        e.preventDefault()

        const pasted = e.clipboardData
            .getData("text")
            .replace(/\D/g, "")
            .slice(0, 5)

        pasted.split("").forEach((digit, index) => {
            const input = verificationRefs.current[index]

            if (input)
                input.value = digit
        })

        const nextIndex = Math.min(pasted.length, 4)

        verificationRefs.current[nextIndex]?.focus()
    }
    const [captchaData, setCaptchaData] = useState<{
        masterImage: string
        titleImage: string,
        dx: number,
        dy: number
    } | null>(null)
    useEffect(() => {
        const v = registerRef.current
        const vPass = passwordRef.current
        const vUser = userNameRef.current
        if (v) {
            if (v.textContent == "") {
                v.textContent = "Email"
            }
            v.addEventListener("focusin", () => {
                v.textContent = ""
            })


            v.addEventListener("focusout", () => {
                if (v.textContent == "") {
                    v.textContent = "Email"
                }
            })

            v.addEventListener("keydown", (k) => {
                if (k.key == "Enter")
                    k.preventDefault()
            })
        }
        if (vPass) {
            if (vPass.textContent == "") {
                vPass.textContent = "Password"
            }

            vPass.addEventListener("focusin", () => {
                vPass.textContent = ""
            })


            vPass.addEventListener("focusout", () => {
                if (vPass.textContent == "")
                    vPass.textContent = "Password"
            })

            vPass.addEventListener("keydown", (k) => {
                if (k.key == "Enter")
                    k.preventDefault()
            })
        }
        if (vUser) {
            if (vUser.textContent == "")
                vUser.textContent = "Username"

            vUser.addEventListener("focusin", () => {
                vUser.textContent = ""
            })


            vUser.addEventListener("focusout", () => {
                if (vUser.textContent == "") {
                    vUser.textContent = "Username"
                }
            })

            vUser.addEventListener("keydown", (k) => {
                if (k.key == "Enter")
                    k.preventDefault()
            })
        }

        const query = submitRegister.current
        const backiRooti = captchaBackground.current
        const itsMenu = captchaMenu.current
        query?.addEventListener('click', async () => {
            let captcha
            if (loc.pathname == "/register")
                captcha = await register()
            else if (loc.pathname == "/login")
                captcha = await login()

            if (captcha) {
                if (captcha.status != 202)
                    return

                setCaptchaData({
                    masterImage: captcha.masterImage,
                    titleImage: captcha.titleImage,
                    dx: captcha.dx,
                    dy: captcha.dy
                })

                rooti?.appendChild(backiRooti!)
                rooti?.appendChild(itsMenu!)
                backiRooti!.style.display = "block"
                itsMenu!.style.display = "block"
            }
        })
        backiRooti?.addEventListener('click', () => {
            backiRooti!.style.display = "none"
            itsMenu!.style.display = "none"
        })
    })
    switch (loc.pathname) {
        case "/":
            Elementa = <div className="middle">
                <div>

                </div>
                <div>

                </div>
            </div>
            break
        case "/register":
            Elementa = <div className="register">
                <div className="register-form-card">

                    <div className="register-form-title">
                        Create your account
                    </div>
                    <br></br>
                    <div className="register-fields">

                        <div
                            className="register-field"
                            contentEditable="true"
                            data-placeholder="Email"
                            role="textbox"
                            ref={registerRef}
                            suppressContentEditableWarning={true}
                        />

                        <div
                            className="register-field"
                            contentEditable="true"
                            data-placeholder="Username"
                            role="textbox"
                            ref={userNameRef}
                            suppressContentEditableWarning={true}
                        />

                        <div
                            className="register-field"
                            contentEditable="true"
                            data-placeholder="Password"
                            role="textbox"
                            ref={passwordRef}
                            suppressContentEditableWarning={true}
                        />
                    </div>

                    <div className="register-submit" ref={submitRegister}>
                        Create account
                    </div>

                    <div className="register-divider">
                        <span></span>
                        <p>or continue with</p>
                        <span></span>
                    </div>

                    <div className="register-oauth">

                        <div className="register-oauth-button">
                            <svg viewBox="0 0 24 24" aria-hidden="true">
                                <path
                                    fill="currentColor"
                                    d="M12 0C5.37 0 0 5.37 0 12c0 5.3 3.44 9.8 8.2 11.38.6.11.82-.26.82-.58v-2.05c-3.34.73-4.04-1.42-4.04-1.42-.55-1.39-1.33-1.76-1.33-1.76-1.09-.75.08-.73.08-.73 1.2.09 1.84 1.24 1.84 1.24 1.07 1.83 2.8 1.3 3.48.99.11-.78.42-1.3.76-1.6-2.67-.3-5.47-1.34-5.47-5.93 0-1.31.47-2.38 1.24-3.22-.12-.3-.54-1.52.12-3.17 0 0 1.01-.32 3.3 1.23.96-.27 1.98-.4 3-.4s2.04.13 3 .4c2.29-1.55 3.3-1.23 3.3-1.23.66 1.65.24 2.87.12 3.17.77.84 1.24 1.91 1.24 3.22 0 4.6-2.8 5.62-5.48 5.92.43.37.81 1.1.81 2.22v3.29c0 .32.22.69.83.57A12.01 12.01 0 0 0 24 12c0-6.63-5.37-12-12-12z"
                                />
                            </svg>
                            GitHub
                        </div>

                        <div className="register-oauth-button">
                            <svg viewBox="0 0 24 24" aria-hidden="true">
                                <path
                                    fill="#4285F4"
                                    d="M21.35 12.27c0-.78-.07-1.53-.22-2.27H12v4.3h5.24a4.48 4.48 0 0 1-1.94 2.94v2.45h3.14c1.84-1.69 2.91-4.18 2.91-7.42z"
                                />
                                <path
                                    fill="#34A853"
                                    d="M12 21.99c2.63 0 4.84-.87 6.45-2.36l-3.14-2.45c-.87.58-1.98.92-3.31.92-2.54 0-4.69-1.72-5.46-4.03H3.3v2.53A9.74 9.74 0 0 0 12 21.99z"
                                />
                                <path
                                    fill="#FBBC05"
                                    d="M6.54 14.07a5.86 5.86 0 0 1 0-3.74V7.8H3.3a10 10 0 0 0 0 8.8l3.24-2.53z"
                                />
                                <path
                                    fill="#EA4335"
                                    d="M12 6.3c1.43 0 2.72.49 3.73 1.45l2.8-2.8C16.84 3.38 14.63 2.5 12 2.5a9.74 9.74 0 0 0-8.7 5.3l3.24 2.53C7.31 8.02 9.46 6.3 12 6.3z"
                                />
                            </svg>
                            Google
                        </div>

                    </div>

                </div>
            </div>
            break
        case "/register/validation/jwt":
            if (verification)
                Elementa = <div className="verification">
                    <div>
                        We've sent you a code to your mail
                    </div>

                    <div className="verification-code">
                        {Array.from({ length: 5 }, (_, index) => (
                            <input
                                key={index}
                                ref={(element) => {
                                    verificationRefs.current[index] = element
                                }}
                                type="text"
                                maxLength={1}
                                inputMode="numeric"
                                onChange={(e) =>
                                    handleVerificationInput(e, index)
                                }
                                onKeyDown={(e) =>
                                    handleVerificationKeyDown(e, index)
                                }
                                onPaste={handleVerificationPaste}
                            />
                        ))}
                    </div>

                    <button
                        onClick={async () => {
                            if (verificationRefs.current) {
                                const inputi = verificationRefs.current[0]!.value +
                                    verificationRefs.current[1]!.value + verificationRefs.current[2]!.value +
                                    verificationRefs.current[3]!.value + verificationRefs.current[4]!.value
                                const res = await registerValidationSubmit(inputi)
                                if (!res)
                                    console.log("Success")
                                else
                                    console.log("Failure")
                            }
                        }}
                        className="verification-submit">
                        Submit
                    </button>
                </div>
            break
        case "/login":
            Elementa = <div className="register">
                <div className="register-form-card">

                    <div className="register-form-title">
                        Sign in to your account
                    </div>
                    <br></br>
                    <div className="register-fields">

                        <div
                            className="register-field"
                            contentEditable="true"
                            data-placeholder="Email"
                            role="textbox"
                            ref={registerRef}
                            suppressContentEditableWarning={true}
                        />

                        <div
                            className="register-field"
                            contentEditable="true"
                            data-placeholder="Password"
                            role="textbox"
                            ref={passwordRef}
                            suppressContentEditableWarning={true}
                        />
                    </div>

                    <div className="register-submit" ref={submitRegister}>
                        Login
                    </div>

                    <div className="register-divider">
                        <span></span>
                        <p>or continue with</p>
                        <span></span>
                    </div>

                    <div className="register-oauth">

                        <div className="register-oauth-button">
                            <svg viewBox="0 0 24 24" aria-hidden="true">
                                <path
                                    fill="currentColor"
                                    d="M12 0C5.37 0 0 5.37 0 12c0 5.3 3.44 9.8 8.2 11.38.6.11.82-.26.82-.58v-2.05c-3.34.73-4.04-1.42-4.04-1.42-.55-1.39-1.33-1.76-1.33-1.76-1.09-.75.08-.73.08-.73 1.2.09 1.84 1.24 1.84 1.24 1.07 1.83 2.8 1.3 3.48.99.11-.78.42-1.3.76-1.6-2.67-.3-5.47-1.34-5.47-5.93 0-1.31.47-2.38 1.24-3.22-.12-.3-.54-1.52.12-3.17 0 0 1.01-.32 3.3 1.23.96-.27 1.98-.4 3-.4s2.04.13 3 .4c2.29-1.55 3.3-1.23 3.3-1.23.66 1.65.24 2.87.12 3.17.77.84 1.24 1.91 1.24 3.22 0 4.6-2.8 5.62-5.48 5.92.43.37.81 1.1.81 2.22v3.29c0 .32.22.69.83.57A12.01 12.01 0 0 0 24 12c0-6.63-5.37-12-12-12z"
                                />
                            </svg>
                            GitHub
                        </div>

                        <div className="register-oauth-button">
                            <svg viewBox="0 0 24 24" aria-hidden="true">
                                <path
                                    fill="#4285F4"
                                    d="M21.35 12.27c0-.78-.07-1.53-.22-2.27H12v4.3h5.24a4.48 4.48 0 0 1-1.94 2.94v2.45h3.14c1.84-1.69 2.91-4.18 2.91-7.42z"
                                />
                                <path
                                    fill="#34A853"
                                    d="M12 21.99c2.63 0 4.84-.87 6.45-2.36l-3.14-2.45c-.87.58-1.98.92-3.31.92-2.54 0-4.69-1.72-5.46-4.03H3.3v2.53A9.74 9.74 0 0 0 12 21.99z"
                                />
                                <path
                                    fill="#FBBC05"
                                    d="M6.54 14.07a5.86 5.86 0 0 1 0-3.74V7.8H3.3a10 10 0 0 0 0 8.8l3.24-2.53z"
                                />
                                <path
                                    fill="#EA4335"
                                    d="M12 6.3c1.43 0 2.72.49 3.73 1.45l2.8-2.8C16.84 3.38 14.63 2.5 12 2.5a9.74 9.74 0 0 0-8.7 5.3l3.24 2.53C7.31 8.02 9.46 6.3 12 6.3z"
                                />
                            </svg>
                            Google
                        </div>

                    </div>

                </div>
            </div>
            break
        case "/login/validation/jwt":
            if (verification)
                Elementa = <div className="verification">
                    <div>
                        We've sent you a code to your mail
                    </div>

                    <div className="verification-code">
                        {Array.from({ length: 5 }, (_, index) => (
                            <input
                                key={index}
                                ref={(element) => {
                                    verificationRefs.current[index] = element
                                }}
                                type="text"
                                maxLength={1}
                                inputMode="numeric"
                                onChange={(e) =>
                                    handleVerificationInput(e, index)
                                }
                                onKeyDown={(e) =>
                                    handleVerificationKeyDown(e, index)
                                }
                                onPaste={handleVerificationPaste}
                            />
                        ))}
                    </div>

                    <button
                        onClick={async () => {
                            if (verificationRefs.current) {
                                const inputi = verificationRefs.current[0]!.value +
                                    verificationRefs.current[1]!.value + verificationRefs.current[2]!.value +
                                    verificationRefs.current[3]!.value + verificationRefs.current[4]!.value
                                const res = await loginValidationSubmit(inputi)
                                if (!res)
                                    console.log("Success")
                                else
                                    console.log("Failure")
                            }
                        }}
                        className="verification-submit">
                        Submit
                    </button>
                </div>
            break
        case "/forgetPassword":
            break
        case "/forgetPassword/validate":
            break
        case "/forgetPassword/validate/jwt":
            break
        case "/forgetPassword/" + params.key:
            break
        case "/posts":
            break
        case "/post/" + params.postid:
            break
        case "/dashboard":
            break
        default:
            Elementa = <div>404 - Page Not Found</div>
            break
    }

    return (<>
        <div className="captchauation" ref={captchaBackground}>
        </div>
        <div className="menuitself" ref={captchaMenu}>
            {captchaData && (
                <Captcha
                    masterImage={captchaData.masterImage}
                    titleImage={captchaData.titleImage}
                    dx={captchaData.dx}
                    dy={captchaData.dy}
                    onSubmit={validationFucker}
                    registerRef={registerRef}
                    userNameRef={userNameRef}
                    passwordRef={passwordRef}
                    loc={loc.pathname}
                />
            )}
        </div>
        {Elementa!}
    </>)
}