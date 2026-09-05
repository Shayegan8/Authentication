import { useLocation, useParams } from "react-router-dom"

export default function Middle() {
    const loc = useLocation()
    const params = useParams()
    switch (loc.pathname) {
        case "/":
            break
        case "/register":
            break
        case "/register/validation":
            break
        case "/register/validation/jwt":
            break
        case "/register/validation/submit":
            break
        case "/login":
            break
        case "/login/validation":
            break
        case "/login/validation/jwt":
            break
        case "/login/validation/submit":
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
            break
    }

    return (
        <>
        </>
    )
}