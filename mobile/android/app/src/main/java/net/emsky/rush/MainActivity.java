package net.emsky.rush;

import android.app.Activity;
import android.os.Bundle;
import android.view.View;
import android.view.WindowManager;
import android.widget.TextView;
import android.widget.FrameLayout;
import android.view.Gravity;
import android.os.Handler;
import android.os.Looper;

import androidx.appcompat.app.AppCompatActivity;
import androidx.core.view.WindowCompat;
import androidx.core.view.WindowInsetsCompat;
import androidx.core.view.WindowInsetsControllerCompat;
import net.emsky.rush.mobile.EbitenView;
import go.Seq;

/**
 * 主活动类
 * 负责启动和初始化 Ebiten 游戏
 */
public class MainActivity extends AppCompatActivity {

    private Handler handler = new Handler(Looper.getMainLooper());
    private Runnable checkExitRunnable;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        // 重要：初始化 Go 序列化上下文
        Seq.setContext(getApplicationContext());

        // 设置全屏模式
        setFullscreenMode();

        // 启动 Ebiten 游戏
        try {
            EbitenView ebitenView = new EbitenView(this);
            setContentView(ebitenView);

            // 启动定期检查退出状态的线程
            startExitCheck();

        } catch (Exception e) {
            e.printStackTrace();
            // 如果初始化失败，显示错误信息
            setContentView(R.layout.activity_main);
        }
    }

    private void setFullscreenMode() {
        // 设置全屏标志
        getWindow().setFlags(
            WindowManager.LayoutParams.FLAG_FULLSCREEN,
            WindowManager.LayoutParams.FLAG_FULLSCREEN
        );

        // 隐藏系统UI
        View decorView = getWindow().getDecorView();
        int uiOptions = View.SYSTEM_UI_FLAG_FULLSCREEN
                | View.SYSTEM_UI_FLAG_HIDE_NAVIGATION
                | View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY
                | View.SYSTEM_UI_FLAG_LAYOUT_STABLE
                | View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION
                | View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN;
        decorView.setSystemUiVisibility(uiOptions);

        // 使用 WindowInsetsController 隐藏系统栏（Android 11+）
        WindowInsetsControllerCompat controller = new WindowInsetsControllerCompat(getWindow(), getWindow().getDecorView());
        controller.hide(WindowInsetsCompat.Type.systemBars());
        controller.setSystemBarsBehavior(WindowInsetsControllerCompat.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE);
    }

    private void startExitCheck() {
        checkExitRunnable = new Runnable() {
            @Override
            public void run() {
                // 检查是否需要退出应用
                if (shouldExitApp()) {
                    finish();
                    return;
                }
                // 每100ms检查一次
                handler.postDelayed(this, 100);
            }
        };
        handler.post(checkExitRunnable);
    }

    private boolean shouldExitApp() {
        try {
            // 调用 Go 代码检查退出标志
            return net.emsky.rush.mobile.Mobile.shouldExit();
        } catch (Exception e) {
            return false;
        }
    }

    @Override
    protected void onDestroy() {
        super.onDestroy();
        if (checkExitRunnable != null) {
            handler.removeCallbacks(checkExitRunnable);
        }
    }

    @Override
    public void onWindowFocusChanged(boolean hasFocus) {
        super.onWindowFocusChanged(hasFocus);
        if (hasFocus) {
            setFullscreenMode();
        }
    }
}