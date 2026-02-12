declare module 'guacamole-common-js' {
    export class Mouse {
        constructor(element: HTMLElement);
        onmousedown: (state: MouseState) => void;
        onmouseup: (state: MouseState) => void;
        onmousemove: (state: MouseState) => void;
    }

    export class Keyboard {
        constructor(element: HTMLElement | Document);
        onkeydown: (keysym: number) => void;
        onkeyup: (keysym: number) => void;
    }

    export class ArrayBufferReader {
        constructor(stream: InputStream);
        onbody: (data: ArrayBuffer) => void;
        onend: () => void;
    }

    export class ArrayBufferWriter {
        constructor(stream: OutputStream);
        sendData(data: ArrayBuffer): void;
        onack: (status: Status) => void;
    }

    export class StringReader {
        constructor(stream: InputStream);
        ontext: (text: string) => void;
        onend: () => void;
    }

    export class StringWriter {
        constructor(stream: OutputStream);
        sendText(text: string): void;
        onack: (status: Status) => void;
    }

    export class InputStream {
        onblob: (data: string) => void;
        onend: () => void;
    }

    export class OutputStream {
        onack: (status: Status) => void;
    }

    export class Status {
        code: number;
        message: string;
        isError(): boolean;
    }

    export class WebSocketTunnel {
        constructor(url: string);
        connect(data: string): void;
        onstatechange: (state: number) => void;
        onerror: (status: Status) => void;
        state: number;
    }

    export class HTTPTunnel {
        constructor(url: string);
        connect(data: string): void;
        onstatechange: (state: number) => void;
        onerror: (status: Status) => void;
        state: number;
    }

    export class Client {
        constructor(tunnel: WebSocketTunnel | HTTPTunnel);
        connect(data?: string): void;
        disconnect(): void;
        sendSize(width: number, height: number): void;
        sendKeyEvent(pressed: number, keysym: number): void;
        sendMouseState(mouseState: MouseState): void;
        getDisplay(): Display;

        onstatechange: (state: number) => void;
        onerror: (status: Status) => void;
        onname: (name: string) => void;
        onclipboard: (stream: InputStream, mimetype: string) => void;
    }

    export class Display {
        getElement(): HTMLElement;
        getWidth(): number;
        getHeight(): number;
    }

    export interface MouseState {
        x: number;
        y: number;
        left: boolean;
        middle: boolean;
        right: boolean;
        up: boolean;
        down: boolean;
    }
}
