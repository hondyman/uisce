import { Kernel } from "./kernel/Kernel"
import { StudioShell } from "./core/StudioShell"
import "./studio.css"

const kernel = new Kernel()
kernel.start()

ReactDOM.render(<StudioShell kernel={kernel} />, document.getElementById("root"))