package ir.tafakoritech.huginn.feature.splash

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import ir.tafakoritech.huginn.R

/** Compose splash shown while the session is being restored. */
@Composable
fun SplashScreen(modifier: Modifier = Modifier) {
    val colors = androidx.compose.material3.MaterialTheme.colorScheme
    Column(
        modifier = modifier
            .fillMaxSize()
            .background(ir.tafakoritech.huginn.core.theme.huginnColors().background),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Image(
            painter = painterResource(R.drawable.ic_logo),
            contentDescription = null,
            modifier = Modifier.size(96.dp),
        )
        Spacer(Modifier.height(20.dp))
        Text(
            text = stringResource(R.string.app_name),
            style = MaterialTheme.typography.headlineMedium,
            color = ir.tafakoritech.huginn.core.theme.huginnColors().onSurface,
        )
        Spacer(Modifier.height(4.dp))
        Text(
            text = stringResource(R.string.brand_subtitle),
            style = MaterialTheme.typography.bodyMedium,
            color = ir.tafakoritech.huginn.core.theme.huginnColors().mutedText,
        )
    }
}
