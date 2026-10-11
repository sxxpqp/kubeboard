package api

import (
	"bufio"
	"context"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// currentUser 返回当前请求已认证的用户名（中间件注入）。
func currentUser(c *gin.Context) string {
	if v, ok := c.Get("username"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "unknown"
}

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	Subprotocols:    []string{"v4.channel.k8s.io", "channel.k8s.io"},
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// podLogs 流式返回 Pod 日志（SSE 风格，纯文本 chunked）
func (h *Handler) podLogs(c *gin.Context) {
	namespace := c.Query("namespace")
	name := c.Query("name")
	container := c.Query("container")
	tailLines := int64(500)
	if v := c.Query("tailLines"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			tailLines = n
		}
	}
	follow := c.Query("follow") == "true"

	opts := &corev1.PodLogOptions{
		Container: container,
		TailLines: &tailLines,
		Follow:    follow,
	}
	req := h.k8s.Clientset.CoreV1().Pods(namespace).GetLogs(name, opts)
	stream, err := req.Stream(c.Request.Context())
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	defer stream.Close()

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)

	flusher, _ := c.Writer.(http.Flusher)
	reader := bufio.NewReader(stream)
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			if _, werr := c.Writer.Write(buf[:n]); werr != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			return
		}
	}
}

// 允许的交互式 shell 白名单，避免 URL 参数变成任意命令入口。
var allowedShells = map[string]bool{
	"/bin/sh":   true,
	"/bin/bash": true,
	"/bin/ash":  true,
	"/bin/zsh":  true,
}

// podTerminal WebSocket 交互式终端
func (h *Handler) podTerminal(c *gin.Context) {
	namespace := c.Query("namespace")
	pod := c.Query("name")
	container := c.Query("container")
	shell := c.DefaultQuery("shell", "/bin/sh")
	if !allowedShells[shell] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "shell not allowed: " + shell})
		return
	}

	log.Printf("[exec] user=%s pod=%s/%s container=%s shell=%s ip=%s",
		currentUser(c), namespace, pod, container, shell, c.ClientIP())

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	req := h.k8s.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(pod).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   []string{shell},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(h.k8s.RestConfig, "POST", req.URL())
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("建立执行器失败: "+err.Error()+"\r\n"))
		return
	}

	stdin := newWSReader(conn)
	stdout := newWSWriter(conn)

	err = exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stdout,
		Tty:    true,
	})
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("\r\n会话结束: "+err.Error()+"\r\n"))
	}
}

type wsReader struct {
	conn   *websocket.Conn
	reader io.Reader
}

func newWSReader(conn *websocket.Conn) *wsReader { return &wsReader{conn: conn} }

func (w *wsReader) Read(p []byte) (int, error) {
	for {
		if w.reader != nil {
			n, err := w.reader.Read(p)
			if err == io.EOF {
				w.reader = nil
				if n > 0 {
					return n, nil
				}
				continue
			}
			return n, err
		}
		_, msg, err := w.conn.ReadMessage()
		if err != nil {
			return 0, err
		}
		w.reader = &sliceReader{data: msg}
	}
}

type sliceReader struct {
	data []byte
	pos  int
}

func (s *sliceReader) Read(p []byte) (int, error) {
	if s.pos >= len(s.data) {
		return 0, io.EOF
	}
	n := copy(p, s.data[s.pos:])
	s.pos += n
	return n, nil
}

type wsWriter struct{ conn *websocket.Conn }

func newWSWriter(conn *websocket.Conn) *wsWriter { return &wsWriter{conn: conn} }

func (w *wsWriter) Write(p []byte) (int, error) {
	if err := w.conn.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}
